// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package single

import (
	"strings"

	"github.com/spf13/pflag"

	comerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
)

// Config is the defaults configuration that can be configured using the repository configuration for the test-single command
type Config struct {
	FilePath string `json:"testrunPath"`
}

func (t *test) getConfig(ghClient github.Client, flagset *pflag.FlagSet) (*Config, error) {
	var cfg Config
	if err := ghClient.GetConfig(t.Command(), &cfg); err != nil {
		if !comerrors.IsNotFound(err) {
			t.log.Error(err, "unable to get default config")
			return nil, pluginerr.New("Unable to read the default config", "The TM Bot was unable to get the default config from the repository")
		}
	}

	if flagset.Arg(0) != "" {
		cutset := " \r"
		cfg.FilePath = strings.Trim(flagset.Arg(0), cutset)
	}

	if cfg.FilePath == "" && flagset.NArg() != 1 {
		return nil, pluginerr.New(`No path to a testrun was specified.
Maybe check whether a bot configuration with a default test has been specified in <RepoRoot>/.ci/tm-config.yaml.`, "")
	}

	return &cfg, nil
}
