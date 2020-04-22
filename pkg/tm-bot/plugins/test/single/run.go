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
	"context"
	"fmt"
	"strings"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	testutil "github.com/gardener/test-infra/pkg/tm-bot/plugins/test"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
	"github.com/ghodss/yaml"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/spf13/pflag"
)

func (t *test) Run(flagset *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error {
	logger := t.log.WithValues("owner", event.GetOwnerName(), "repo", event.GetRepositoryName(), "runID", t.runID)
	ctx := context.Background()
	defer ctx.Done()

	cfg, err := t.getConfig(client, flagset)
	if err != nil {
		return err
	}

	content, err := client.GetContent(ctx, event, cfg.FilePath)
	if err != nil {
		logger.Error(err, "unable to get content of file", "path", cfg.FilePath)
		return pluginerr.New(fmt.Sprintf("Sorry, but I was unable to redner the Testrun from the file at %s.", cfg.FilePath), "Unable to get the content of the specified file.")
	}

	tr, err := testmachinery.ParseTestrun(content)
	if err != nil {
		logger.Error(err, "unable to parse testrun", "path", cfg.FilePath)
		return pluginerr.New(fmt.Sprintf("Sorry, but I was unable to redner the Testrun from the file at %s.", cfg.FilePath), "The Testrun could not be parsed.")
	}

	tr.GenerateName = fmt.Sprintf("bot-%s-", t.Command())
	tr.Name = ""

	if err := testutil.InjectRepositoryLocation(event, tr); err != nil {
		logger.Error(err, "unable to inject current repository")
		return pluginerr.New(fmt.Sprintf("Sorry, but I was unable to render the Testrun from the file at %s.", cfg.FilePath), "Current repository could not be injected.")
	}

	if t.dryRun {
		stepsTable := &strings.Builder{}
		output.RenderTestflowTable(stepsTable, tr.Spec.TestFlow)
		_, err := client.Comment(ctx, event, plugins.FormatResponseWithReason(event.GetAuthorName(),
			fmt.Sprintf("I rendered the testrun for you.\nView the full test in the details section.\n<pre>%s</pre>", stepsTable.String()),
			fmt.Sprintf("<pre>%s</pre>", util.PrettyPrintStruct(tr))))
		return err
	}

	tr, updater, err := t.runs.CreateTestrun(logger, ctx, client, event, tr)
	if err != nil {
		return err
	}

	var state interface{} = testutil.State{
		TestrunID: tr.Name,
		Namespace: tr.Namespace,
		CommentID: updater.GetCommentID(),
	}
	stateByte, err := yaml.Marshal(state)
	if err := plugins.UpdateState(t, t.runID, string(stateByte)); err != nil {
		logger.Error(err, "unable to persist state")
	}

	_, err = t.runs.Watch(logger, ctx, updater, event, tr, t.interval, t.timeout)
	if err != nil {
		return err
	}

	return nil
}

func (t *test) ResumeFromState(client github.Client, event *github.GenericRequestEvent, stateString string) error {
	logger := t.log.WithValues("owner", event.GetOwnerName(), "repo", event.GetRepositoryName(), "runID", t.runID)
	ctx := context.Background()
	defer ctx.Done()
	state := testutil.State{}
	if err := yaml.Unmarshal([]byte(stateString), &state); err != nil {
		t.log.Error(err, "unable to parse state")
		return pluginerr.NewRecoverable("unable to recover state", err.Error())
	}

	tr := &v1beta1.Testrun{
		ObjectMeta: v1.ObjectMeta{
			Name:      state.TestrunID,
			Namespace: state.Namespace,
		},
	}
	updater := tests.NewStatusUpdaterFromCommentID(logger, client, event, state.CommentID)

	_, err := t.runs.Watch(logger, ctx, updater, event, tr, t.interval, t.timeout)
	if err != nil {
		return err
	}

	return nil
}
