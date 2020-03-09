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

package skip

import (
	"context"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
)

type skip struct {
	log   logr.Logger
	runID string
}

func New(log logr.Logger) plugins.Plugin {
	return &skip{log: log}
}

func (e *skip) New(runID string) plugins.Plugin {
	return &skip{
		log:   e.log,
		runID: runID,
	}
}

func (_ *skip) Command() string {
	return "skip"
}

func (_ *skip) Authorization() github.AuthorizationType {
	return github.AuthorizationCodeOwners
}

func (_ *skip) Description() string {
	return "Clears the testmachinery github status to skip a failed or pending test"
}

func (_ *skip) Example() string {
	return "/skip"
}

func (_ *skip) Config() string {
	return ""
}

func (_ *skip) ResumeFromState(_ github.Client, _ *github.GenericRequestEvent, _ string) error {
	return nil
}

func (e *skip) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(e.Command(), pflag.ContinueOnError)
	return flagset
}

func (e *skip) Run(flagset *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error {
	ctx := context.Background()
	defer ctx.Done()

	updater := tests.NewStatusUpdater(e.log, client, event)
	if err := updater.UpdateStatus(context.TODO(), github.StateSuccess, "skipped"); err != nil {
		e.log.Error(err, "unable to update github status to skipped")
		_, err = client.Comment(ctx, event, plugins.FormatErrorResponse(event.GetAuthorName(), "I was unable to skip the tests. Please try again", ""))
		return err
	}

	_, err := client.Comment(ctx, event, plugins.FormatSimpleResponse(event.GetAuthorName(), "I skipped all necessary testmachinery tests"))
	return err
}
