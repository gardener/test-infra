// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package options

import (
	"fmt"

	cdoci "github.com/gardener/component-spec/bindings-go/oci"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/pflag"

	"github.com/gardener/component-cli/ociclient"
	"github.com/gardener/component-cli/ociclient/cache"
	"github.com/gardener/component-cli/ociclient/credentials"
	"github.com/gardener/component-cli/ociclient/credentials/secretserver"
)

// OCIOptions defines a set of options to create a oci client
type Options struct {
	// AllowPlainHttp allows the fallback to http if the oci registry does not support https
	AllowPlainHttp bool
	// CacheDir defines the oci cache directory
	CacheDir string
	// RegistryConfigPath defines a path to the dockerconfig.json with the oci registry authentication.
	RegistryConfigPath string
	// ConcourseConfigPath is the path to the local concourse config file.
	ConcourseConfigPath string
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	if fs == nil {
		fs = pflag.CommandLine
	}

	fs.BoolVar(&o.AllowPlainHttp, "allow-plain-http", false, "allows the fallback to http if the oci registry does not support https")
	fs.StringVar(&o.RegistryConfigPath, "registry-config", "", "path to the dockerconfig.json with the oci registry authentication information")
	fs.StringVar(&o.ConcourseConfigPath, "cc-config", "", "path to the local concourse config file")
}

// Builds a new oci client based on the given options
func (o *Options) Build(log logr.Logger, fs vfs.FileSystem) (ociclient.Client, cache.Cache, error) {
	cache, err := cache.NewCache(log, cache.WithBasePath(o.CacheDir))
	if err != nil {
		return nil, nil, err
	}

	ociOpts := []ociclient.Option{
		ociclient.WithCache{Cache: cache},
		ociclient.WithKnownMediaType(cdoci.ComponentDescriptorConfigMimeType),
		ociclient.WithKnownMediaType(cdoci.ComponentDescriptorTarMimeType),
		ociclient.WithKnownMediaType(cdoci.ComponentDescriptorJSONMimeType),
		ociclient.AllowPlainHttp(o.AllowPlainHttp),
	}
	if len(o.RegistryConfigPath) != 0 {
		keyring, err := credentials.CreateOCIRegistryKeyring(nil, []string{o.RegistryConfigPath})
		if err != nil {
			return nil, nil, fmt.Errorf("unable to create keyring for registry at %q: %w", o.RegistryConfigPath, err)
		}
		ociOpts = append(ociOpts, ociclient.WithKeyring(keyring))
	} else {
		keyring, err := secretserver.New().
			WithFS(fs).
			FromPath(o.ConcourseConfigPath).
			WithMinPrivileges(secretserver.ReadWrite).
			Build()
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get credentils from secret server: %s", err.Error())
		}
		if keyring != nil {
			ociOpts = append(ociOpts, ociclient.WithKeyring(keyring))
		}
	}

	ociClient, err := ociclient.NewClient(log, ociOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to build oci client: %w", err)
	}
	return ociClient, cache, nil
}
