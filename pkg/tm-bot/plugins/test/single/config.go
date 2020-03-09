// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package single

import (
	comerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/spf13/pflag"
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
			return nil, errors.New("Unable to read the default config", "The TM Bot was unable to get the default config from the repository")
		}
	}

	if cfg.FilePath == "" && flagset.NArg() != 1 {
		return nil, errors.New("No path to a testrun was specified", "")
	}

	cfg.FilePath = flagset.Arg(0)
	return &cfg, nil
}
