// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package common

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
)

type test struct {
	runID string
	log   logr.Logger

	runs     *tests.Runs
	timeout  time.Duration
	interval time.Duration

	dryRun     bool
	testConfig tests.TestConfig
}

func New(log logr.Logger, runs *tests.Runs) plugins.Plugin {
	return &test{
		log:      log.WithName("common-test"),
		runs:     runs,
		timeout:  5 * time.Hour,
		interval: 1 * time.Minute,
	}
}

func (t *test) New(runID string) plugins.Plugin {
	return &test{
		runID:    runID,
		log:      t.log,
		runs:     t.runs,
		timeout:  t.timeout,
		interval: t.interval,
	}
}

func (t *test) Command() string {
	return "test"
}

func (_ *test) Authorization() github.AuthorizationType {
	return github.AuthorizationTeam
}

func (t *test) Description() string {
	return `Runs a testrun that is either specified by command line flags or in the default values.
The specified path is rendered as testrun and the current repository is injected as a default location.
`
}

func (t *test) Example() string {
	return fmt.Sprintf("/%s [sub-command] [--flags]", t.Command())
}

func (t *test) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(t.Command(), pflag.ContinueOnError)

	flagset.StringVar(&t.testConfig.FilePath, "testrunPath", "", "path to the testrun file that should be executed")
	flagset.StringArrayVar(&t.testConfig.SetValues, "set", make([]string, 0), "sets additional helm values")
	flagset.BoolVar(&t.testConfig.Template, "template", false, "run go templating on the configured file before execution")
	flagset.BoolVar(&t.dryRun, "dry-run", false, "Print the rendered testrun")
	return flagset
}

func (t *test) Config() string {
	return `
test:
  # Configures the default to run when specifying no additional parameters: /test .
  default: 
    testrunPath: path/to/testrun
    template: <bool> # default to false; Will run go-template

  # Configures an additional subcommand that execute the defined test when running /test <subcommand>.
  # For example the first configured test is executed when commenting a PR with "/test my-subcommand".
  tests:
  # simple testrun execution
  - subCommand: my-subcommand
    testrunPath: path/to/testrun # runs the specified testrun
	template: <bool> # default to false; Will run go-template

  # simple templated testrun
  # configure "template=true" to template the configured testrun with gotemplate
  # and the values configured by "--set". The helm syntax is used to configure parameters.
  - subCommand: templated-test
    testrunPath: path/to/testrun # runs the specified testrun
	template: <bool> # defaults to false;
`
}
