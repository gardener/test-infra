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
	"fmt"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"time"
)

type test struct {
	runID string
	log   logr.Logger

	runs     *tests.Runs
	timeout  time.Duration
	interval time.Duration

	dryRun bool
}

func New(log logr.Logger, runs *tests.Runs) plugins.Plugin {
	return &test{
		log:      log.WithName("test-single"),
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
	return "test-single"
}

func (_ *test) Authorization() github.AuthorizationType {
	return github.AuthorizationTeam
}

func (t *test) Description() string {
	return `Runs a single testrun that is either specified by command line flag or in the default values
The specified path is rendered as testrun and the current repository is injected as a default location.
`
}

func (t *test) Example() string {
	return fmt.Sprintf("/%s [path to the testrun]", t.Command())
}

func (t *test) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(t.Command(), pflag.ContinueOnError)
	flagset.BoolVar(&t.dryRun, "dry-run", false, "Print the rendered testrun")
	return flagset
}

func (t *test) Config() string {
	return `
test-single:
  testrunPath: path/to/testrun
`
}
