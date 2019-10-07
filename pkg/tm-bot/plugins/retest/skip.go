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
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
)

type retest struct {
	log   logr.Logger
	runID string
}

func New(log logr.Logger) plugins.Plugin {
	return &retest{log: log}
}

func (e *retest) New(runID string) plugins.Plugin {
	return &retest{
		log:   e.log,
		runID: runID,
	}
}

func (_ *retest) Command() string {
	return "retest"
}

func (_ *retest) Authorization() github.AuthorizationType {
	return github.AuthorizationOrg
}

func (_ *retest) Description() string {
	return "Retests the last integration test of the Pull request"
}

func (_ *retest) Example() string {
	return "/retest"
}

func (_ *retest) ResumeFromState(_ github.Client, _ *github.GenericRequestEvent, _ string) error {
	return nil
}

func (e *retest) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(e.Command(), pflag.ContinueOnError)
	return flagset
}

func (e *retest) Run(flagset *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error {

	// get the last test command
	issue, err := client.GetIssue(event)
	if err != nil {
		return err
	}
	client.Client().Issues.ListComments()

	_, err := client.Comment(event, plugins.FormatSimpleResponse(event.GetAuthorName(), "I skipped all necessary testmachinery tests"))
	return err
}
