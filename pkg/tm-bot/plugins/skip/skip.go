// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package skip

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
)

type skip struct {
	log   logr.Logger
	runID string
}

func New(log logr.Logger) plugins.Plugin {
	return &skip{log: log}
}

func (s *skip) New(runID string) plugins.Plugin {
	return &skip{
		log:   s.log,
		runID: runID,
	}
}

func (s *skip) Command() string {
	return "skip"
}

func (s *skip) Authorization() github.AuthorizationType {
	return github.AuthorizationCodeOwners
}

func (s *skip) Description() string {
	return "Clears the testmachinery github status to skip a failed or pending test"
}

func (s *skip) Example() string {
	return "/skip"
}

func (s *skip) Config() string {
	return ""
}

func (s *skip) ResumeFromState(_ github.Client, _ *github.GenericRequestEvent, _ string) error {
	return nil
}

func (s *skip) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(s.Command(), pflag.ContinueOnError)
	return flagset
}

func (s *skip) Run(flagset *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error {
	ctx := context.Background()
	defer ctx.Done()

	updater := tests.NewStatusUpdater(s.log, client, event)
	if err := updater.UpdateStatus(context.TODO(), github.StateSuccess, "skipped"); err != nil {
		s.log.Error(err, "unable to update github status to skipped")
		_, err = client.Comment(ctx, event, plugins.FormatErrorResponse(event.GetAuthorName(), "I was unable to skip the tests. Please try again", ""))
		return err
	}

	_, err := client.Comment(ctx, event, plugins.FormatSimpleResponse(event.GetAuthorName(), "I skipped all necessary testmachinery tests"))
	return err
}
