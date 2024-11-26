// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testmachinery

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/gardener/test-infra/pkg/apis/config"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var tmConfig = TmConfiguration{
	Configuration: &config.Configuration{},
}

// Setup fetches all configuration values and creates the TmConfiguration.
func Setup(config *config.Configuration) error {
	tmConfig = TmConfiguration{
		Configuration: config,
	}

	var err error
	tmConfig.GitHubSecrets, err = readSecretsFromFile(config.GitHub.SecretsPath)
	if err != nil {
		return err
	}

	// if no endpoint is defined we assume that no cleanup should happen
	// this should only happen in local environments
	//if tmConfig.TestMachinery.Local && config.S3.Server == nil {
	//	tmConfig.S3 = nil
	//}

	if config.ElasticSearch == nil || len(config.ElasticSearch.Endpoint) == 0 {
		tmConfig.ElasticSearch = nil
	}

	if len(config.TestMachinery.RetryTimeout) != 0 {
		d, err := time.ParseDuration(config.TestMachinery.RetryTimeout)
		if err != nil {
			return fmt.Errorf("unable to parse retry timeout: %w", err)
		}
		config.TestMachinery.RetryTimeoutDuration = &d
	}
	if config.TestMachinery.RetryTimeoutDuration == nil {
		// default timeout location
		d := 15 * time.Minute
		config.TestMachinery.RetryTimeoutDuration = &d
	}

	return nil
}

// GetConfig returns the current testmachinery configuration.
func GetConfig() *TmConfiguration {
	return &tmConfig
}

// GetNamespace returns the current testmachinery namespace.
func GetNamespace() string {
	return tmConfig.TestMachinery.Namespace
}

// CleanWorkflowPods returns whether pod gc is enabled.
func CleanWorkflowPods() bool {
	return tmConfig.TestMachinery.CleanWorkflowPods
}

// TestDefPath returns the path to TestDefinition inside repositories (scripts/integration-tests/argo/tm).
func TestDefPath() string {
	return tmConfig.TestMachinery.TestDefPath
}

// Locations returns the locations configuration
func Locations() config.Locations {
	return tmConfig.TestMachinery.Locations
}

// Prepare Image returns the image of the prepare step.
func PrepareImage() string {
	return tmConfig.TestMachinery.PrepareImage
}

// BaseImage returns the default image that is used if no image is specified by a TestDefinition.
func BaseImage() string {
	return tmConfig.TestMachinery.BaseImage
}

// GetGitHubSecrets returns all github secrets
func GetGitHubSecrets() []GitHubInstanceConfig {
	return tmConfig.GitHubSecrets
}

// GetS3Configuration returns the current s3 configuration
func GetS3Configuration() *config.S3 {
	return tmConfig.S3
}

// GetElasticsearchConfiguration returns the current elasticsearch configuration
func GetElasticsearchConfiguration() *config.ElasticSearch {
	return tmConfig.ElasticSearch
}

// IsRunLocal returns if the testmachinery is currently running locally
func IsRunLocal() bool {
	return tmConfig.TestMachinery.Local
}

// IsRunInsecure returns if the testmachinery is run locally
func IsRunInsecure() bool {
	return tmConfig.TestMachinery.Insecure
}

func GetLandscapeMappings() []config.LandscapeMapping {
	return tmConfig.TestMachinery.LandscapeMappings
}

// GetRetryTimeout returns the retry
func GetRetryTimeout() *time.Duration {
	return tmConfig.TestMachinery.RetryTimeoutDuration
}

// GetWorkflowName returns the workflow name of a testruns
func GetWorkflowName(tr *tmv1beta1.Testrun) string {
	return fmt.Sprintf("%s-wf", tr.Name)
}

// GetPauseTaskName returns the name of the pause step to a corresponding step.
func GetPauseTaskName(name string) string {
	return fmt.Sprintf("%s-pause", name)
}

func readSecretsFromFile(path string) ([]GitHubInstanceConfig, error) {
	if len(path) == 0 {
		return make([]GitHubInstanceConfig, 0), nil
	}
	if _, err := os.Stat(path); err != nil {
		return nil, errors.Wrapf(err, "file %s does not exist", path)
	}
	rawSecrets, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read file from %s", path)
	}
	gitSecrets := GitHubSecrets{}
	err = yaml.Unmarshal(rawSecrets, &gitSecrets)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse git secrets")
	}
	if len(gitSecrets.Secrets) == 0 {
		return nil, errors.New("git secrets are emtpy")
	}
	return gitSecrets.Secrets, nil
}
