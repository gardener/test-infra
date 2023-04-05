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
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"

	"github.com/Masterminds/sprig"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/ghodss/yaml"
	"github.com/spf13/pflag"
	"helm.sh/helm/v3/pkg/strvals"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	testutil "github.com/gardener/test-infra/pkg/tm-bot/plugins/test"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
)

func (t *test) Run(flagset *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error {
	ctx := context.Background()
	defer ctx.Done()
	log := t.log.WithValues("owner", event.GetOwnerName(), "repo", event.GetRepositoryName(), "runID", t.runID)
	subCommand, test, err := t.getConfig(client, flagset)
	if err != nil {
		return err
	}

	content, err := client.GetContent(ctx, event, test.FilePath)
	if err != nil {
		log.Error(err, "unable to get content of file", "path", test.FilePath)
		return pluginerr.Builder().
			WithShortf("Sorry, but I was unable to render the Testrun from the file at %s.", test.FilePath).
			WithLong("Unable to get the content of the specified file.")
	}

	// template if applicable
	if test.Template {
		content, err = templateTest(test, content)
		if err != nil {
			return err
		}
	}

	tr, err := testmachinery.ParseTestrun(content)
	if err != nil {
		log.Error(err, "unable to parse testrun", "path", test.FilePath)
		return pluginerr.Builder().
			WithShortf("Sorry, but I was unable to render the Testrun from the file at %s.<br>", test.FilePath).
			WithLongf("<pre>%s</pre>", string(content)).ShowLong()
	}

	tr.GenerateName = "e2e-"
	tr.Name = ""

	if err := testutil.InjectRepositoryLocation(event, tr); err != nil {
		log.Error(err, "unable to inject current repository")
		return pluginerr.Builder().
			WithShortf("Sorry, but I was unable to render the Testrun from the file at %s.", test.FilePath).
			WithLong("Current repository could not be injected.")
	}

	if t.dryRun {
		stepsTable := &strings.Builder{}
		output.RenderTestflowTable(stepsTable, tr.Spec.TestFlow)
		_, err := client.Comment(ctx, event, plugins.FormatResponseWithReason(event.GetAuthorName(),
			fmt.Sprintf("I rendered the testrun for you.\nView the full test in the details section.\n<pre>%s</pre>", stepsTable.String()),
			fmt.Sprintf("<pre>%s</pre>", util.PrettyPrintStruct(tr))))
		return err
	}

	statusUpdater := tests.NewStatusUpdater(log, client, event)
	statusUpdater.SetGitHubContext(subCommand)
	if err := t.runs.CreateTestrun(ctx, log, statusUpdater, event, tr); err != nil {
		return err
	}

	var state interface{} = testutil.State{
		TestrunID: tr.Name,
		Namespace: tr.Namespace,
		CommentID: statusUpdater.GetCommentID(),
	}
	stateByte, err := yaml.Marshal(state)
	if err != nil {
		return err
	}
	if err := plugins.UpdateState(t, t.runID, string(stateByte)); err != nil {
		log.Error(err, "unable to persist state")
	}

	_, err = t.runs.Watch(log, ctx, statusUpdater, event, tr, t.interval, t.timeout)
	if err != nil {
		return err
	}

	return nil
}

func templateTest(test *tests.TestConfig, testrunBytes []byte) ([]byte, error) {
	values := map[string]interface{}{}
	for _, val := range test.SetValues {
		newSetValues, err := strvals.ParseString(val)
		if err != nil {
			return nil, pluginerr.Wrapf(err, "I was unable to parse the given templating values:<br> `%s`", val)
		}
		values = utils.MergeMaps(values, newSetValues)
	}

	tmpl, err := template.New("testrun").Funcs(sprig.FuncMap()).Parse(string(testrunBytes))
	if err != nil {
		return nil, pluginerr.Wrap(err, "An error occurred during parsing of the testrun as go template.")
	}

	var templatedTestrunBuf bytes.Buffer
	err = tmpl.Execute(&templatedTestrunBuf, map[string]interface{}{
		"Values": values,
	})
	if err != nil {
		return nil, pluginerr.Wrap(err, "An error occurred during templating of the testrun.")
	}
	return templatedTestrunBuf.Bytes(), nil
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
