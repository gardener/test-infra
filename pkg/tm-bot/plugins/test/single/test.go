// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package single

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

func (t *test) Authorization() github.AuthorizationType {
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
  # configures the default to run when specifying no additional parameters: /test
  testrunPath: path/to/testrun
`
}
