// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package single

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/pflag"
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
		return pluginerr.Builder().
			WithShortf("Sorry, but I was unable to render the Testrun from the file at %s.", cfg.FilePath).
			WithLong("Unable to get the content of the specified file.")
	}

	tr, err := testmachinery.ParseTestrun(content)
	if err != nil {
		logger.Error(err, "unable to parse testrun", "path", cfg.FilePath)
		return pluginerr.Builder().
			WithShortf("Sorry, but I was unable to render the Testrun from the file at %s.<br>", cfg.FilePath).
			WithLongf("<pre>%s</pre>", string(content)).ShowLong()
	}

	tr.GenerateName = "e2e-"
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

	statusUpdater := tests.NewStatusUpdater(logger, client, event)
	statusUpdater.SetGitHubContext("single")
	if err := t.runs.CreateTestrun(ctx, logger, statusUpdater, event, tr); err != nil {
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
		logger.Error(err, "unable to persist state")
	}

	_, err = t.runs.Watch(logger, ctx, statusUpdater, event, tr, t.interval, t.timeout)
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
