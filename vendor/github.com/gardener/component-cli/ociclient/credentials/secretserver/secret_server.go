// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package secretserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"

	"github.com/gardener/component-cli/ociclient/credentials"
)

// ContainerRegistryConfigType is the cc secret server container registry config type
const ContainerRegistryConfigType = "container_registry"

// SecretServerEndpointEnvVarName is the name of the envvar that contains the endpoint of the secret server.
const SecretServerEndpointEnvVarName = "SECRETS_SERVER_ENDPOINT"

// SecretServerConcourseConfigEnvVarName is the name of the envvar that contains the name of concourse config.
const SecretServerConcourseConfigEnvVarName = "SECRETS_SERVER_CONCOURSE_CFG_NAME"

type Privilege string

const (
	ReadOnly  Privilege = "readonly"
	ReadWrite Privilege = "readwrite"
)

// SecretServerConfig is the struct that describes the secret server concourse config
type SecretServerConfig struct {
	ContainerRegistry map[string]*ContainerRegistryCredentials `json:"container_registry"`
}

// ContainerRegistryCredentials describes the container registry credentials struct as igven by the cc secrets server.
type ContainerRegistryCredentials struct {
	Username               string    `json:"username"`
	Password               string    `json:"password"`
	Privileges             Privilege `json:"privileges"`
	Host                   string    `json:"host,omitempty"`
	ImageReferencePrefixes []string  `json:"image_reference_prefixes,omitempty"`
}

// KeyringBuilder is a builder that creates a keyring from a concourse config file.
type KeyringBuilder struct {
	fs            vfs.FileSystem
	path          string
	minPrivileges Privilege
	forRef        string
}

// New creates a new keyring builder.
func New() *KeyringBuilder {
	return &KeyringBuilder{}
}

// WithFS configures the builder to use a different filesystem
func (kb *KeyringBuilder) WithFS(fs vfs.FileSystem) *KeyringBuilder {
	kb.fs = fs
	return kb
}

// FromPath configures local concourse config file.
func (kb *KeyringBuilder) FromPath(path string) *KeyringBuilder {
	kb.path = path
	return kb
}

// For configures the builder to only include the config that one reference.
func (kb *KeyringBuilder) For(ref string) *KeyringBuilder {
	kb.forRef = ref
	return kb
}

// WithMinPrivileges configures the builder to only include credentials with a minimal config
func (kb *KeyringBuilder) WithMinPrivileges(priv Privilege) *KeyringBuilder {
	kb.minPrivileges = priv
	return kb
}

// Build creates a oci keyring based on the given configuration.
// It returns nil if now credentials can be found.
func (kb *KeyringBuilder) Build() (*credentials.GeneralOciKeyring, error) {
	keyring := credentials.New()
	if err := kb.Apply(keyring); err != nil {
		return nil, err
	}
	if keyring.Size() == 0 {
		return nil, nil
	}
	return keyring, nil
}

// Apply applies the found configuration to the given keyring.
func (kb *KeyringBuilder) Apply(keyring *credentials.GeneralOciKeyring) error {
	// set defaults
	if kb.fs == nil {
		kb.fs = osfs.New()
	}
	if len(kb.minPrivileges) == 0 {
		kb.minPrivileges = ReadOnly
	}

	// read from local path
	if len(kb.path) != 0 {
		file, err := kb.fs.Open(kb.path)
		if err != nil {
			return fmt.Errorf("unable to load config from %q: %w", kb.path, err)
		}
		defer file.Close()
		return newKeyring(keyring, file, kb.minPrivileges, kb.forRef)
	}

	secSrvEndpoint, ccConfig := os.Getenv(SecretServerEndpointEnvVarName), os.Getenv(SecretServerConcourseConfigEnvVarName)
	if len(secSrvEndpoint) != 0 && len(ccConfig) != 0 {
		body, err := getConfigFromSecretServer(secSrvEndpoint, ccConfig)
		if err != nil {
			return err
		}
		defer body.Close()
		return newKeyring(keyring, body, kb.minPrivileges, kb.forRef)
	}

	return nil
}

func getConfigFromSecretServer(endpoint, configName string) (io.ReadCloser, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse secret server url %q: %w", endpoint, err)
	}
	u.Path = filepath.Join(u.Path, configName)

	res, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("unable to get config from secret server %q: %w", u.String(), err)
	}
	return res.Body, nil
}

// newKeyring creates a new oci keyring from a config given by the reader
// if ref is defined only the credentials that match the ref are put into the keyring.
func newKeyring(keyring *credentials.GeneralOciKeyring, reader io.Reader, minPriv Privilege, ref string) error {
	config := &SecretServerConfig{}
	if err := json.NewDecoder(reader).Decode(config); err != nil {
		return fmt.Errorf("unable to decode config")
	}

	for key, cred := range config.ContainerRegistry {
		if minPriv == ReadWrite && cred.Privileges == ReadOnly {
			continue
		}

		if len(cred.Host) != 0 {
			host, err := url.Parse(cred.Host)
			if err != nil {
				return fmt.Errorf("unable to parse url %q in config %q: %w", cred.Host, key, err)
			}
			err = keyring.AddAuthConfig(host.Host, dockerconfigtypes.AuthConfig{
				Username: cred.Username,
				Password: cred.Password,
			})
			if err != nil {
				return fmt.Errorf("unable to add auth config: %w", err)
			}
		}
		for _, prefix := range cred.ImageReferencePrefixes {
			err := keyring.AddAuthConfig(prefix, dockerconfigtypes.AuthConfig{
				Username: cred.Username,
				Password: cred.Password,
			})
			if err != nil {
				return fmt.Errorf("unable to add auth config: %w", err)
			}
		}
	}

	return nil
}
