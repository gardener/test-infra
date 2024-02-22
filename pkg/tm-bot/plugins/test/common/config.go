// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"

	comerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
)

// Config is the defaults configuration that can be configured using the repository configuration for the test command
type Config struct {
	// Configures the default test that should be executed when no parameters are specified.
	Default *tests.TestConfig `json:"default,omitempty"`
	// Tests defines repo specific test commands that can execute specific tests.
	Tests []SubCommand `json:"tests,omitempty"`
}

type SubCommand struct {
	// SubCommand defines the subcommand that should match the "/test <subCommand>" call
	// to trigger the configured test
	SubCommand       string `json:"subCommand"`
	tests.TestConfig `json:",inline"`
}

const trimset = " \r"

// getConfig returns the subcommand name, test config given by the arguments and config file.
func (t *test) getConfig(ghClient github.Client, flagset *pflag.FlagSet) (string, *tests.TestConfig, error) {
	var repoConfig Config
	if err := ghClient.GetConfig(t.Command(), &repoConfig); err != nil {
		if !comerrors.IsNotFound(err) {
			t.log.Error(err, "unable to get repository config")
			return "", nil, pluginerr.New("Unable to read the config from the repository", "The TM Bot was unable to get the default config from the repository")
		}
	}

	// return the default testrun if no sub command is given
	if flagset.NArg() == 0 {
		defaultConfig := mergeTestConfig(&t.testConfig, repoConfig.Default)
		if err := validateTestConfig(defaultConfig); err != nil {
			if repoConfig.Default == nil {
				return "", nil, pluginerr.Builder().
					WithShortf("%s.<br> Tip: Check whether the test is correctly defined by flags or by the default test config in the repository.", err.Error()).
					WithLong("")
			}
			return "", nil, err
		}
		return "default", defaultConfig, nil
	}

	subCommandName := flagset.Arg(0)
	subCommandName = strings.Trim(subCommandName, trimset)

	if len(subCommandName) == 0 {
		return "", nil, pluginerr.New("The defined subcommand is emtpy", "")
	}

	test := getConfigForSubCommand(repoConfig.Tests, subCommandName)
	if test == nil {
		return "", nil, pluginerr.New(fmt.Sprintf(`No test configuration found for subcommand %s.
Check check the test configuration in the main branch of your repository under "RepoRoot/.ci/tm-config.yaml".`, subCommandName), "")
	}
	if err := validateTestConfig(test); err != nil {
		return "", nil, err
	}
	return subCommandName, mergeTestConfig(&t.testConfig, test), nil
}

func getConfigForSubCommand(tests []SubCommand, name string) *tests.TestConfig {
	for _, test := range tests {
		if test.SubCommand == name {
			return &test.TestConfig
		}
	}
	return nil
}

func validateTestConfig(cfg *tests.TestConfig) error {
	if len(cfg.FilePath) == 0 {
		return pluginerr.New(`No path to a testrun was specified.
Maybe check whether a bot configuration with a default test has been specified in <RepoRoot>/.ci/tm-config.yaml.`, "")
	}

	return nil
}

// mergeTestConfig merges two test configs whereas b overwrites TestConfiguration defined in a.
func mergeTestConfig(a, b *tests.TestConfig) *tests.TestConfig {
	if b == nil {
		return a
	}
	if a == nil {
		return b
	}
	if len(b.FilePath) != 0 {
		a.FilePath = b.FilePath
	}
	if b.Template {
		a.Template = b.Template
	}
	if len(b.SetValues) != 0 {
		a.SetValues = b.SetValues
	}
	return a
}
