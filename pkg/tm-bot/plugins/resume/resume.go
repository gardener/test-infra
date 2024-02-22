// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package resume

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/util"
)

type resume struct {
	runID     string
	log       logr.Logger
	k8sClient kclient.Client
}

func New(log logr.Logger, k8sClient kclient.Client) plugins.Plugin {
	return &resume{
		log:       log,
		k8sClient: k8sClient,
	}
}

func (r *resume) New(runID string) plugins.Plugin {
	return &resume{runID: runID}
}

func (r *resume) Command() string {
	return "resume"
}

func (r *resume) Authorization() github.AuthorizationType {
	return github.AuthorizationTeam
}

func (r *resume) Description() string {
	return "Prints the provided value"
}

func (r *resume) Example() string {
	return "/resume"
}

func (_ *resume) Config() string {
	return ""
}

func (r *resume) ResumeFromState(_ github.Client, _ *github.GenericRequestEvent, _ string) error {
	return nil
}

func (r *resume) Flags() *pflag.FlagSet {
	flagset := pflag.NewFlagSet(r.Command(), pflag.ContinueOnError)
	return flagset
}

func (r *resume) Run(flagset *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error {
	ctx := context.Background()
	defer ctx.Done()
	run, ok := tests.GetRunning(event)
	if !ok {
		if _, err := client.Comment(ctx, event, plugins.FormatSimpleErrorResponse(event.GetAuthorName(), "There are no running tests for this PR")); err != nil {
			r.log.Error(err, "unable to comment to github")
		}
	}
	logger := r.log.WithValues("testrun", run.Testrun.GetName(), "namespace", run.Testrun.GetNamespace())

	tr := &v1beta1.Testrun{}
	if err := r.k8sClient.Get(ctx, kclient.ObjectKey{Name: run.Testrun.Name, Namespace: run.Testrun.Namespace}, tr); err != nil {
		logger.Error(err, "unable to to fetch testrun")
		if _, err := client.Comment(ctx, event, plugins.FormatSimpleErrorResponse(event.GetAuthorName(), "There are no running tests for this PR")); err != nil {
			logger.Error(err, "unable to comment to github")
		}
	}

	if err := util.ResumeTestrun(ctx, r.k8sClient, tr); err != nil {
		logger.Error(err, "unable to resume testrun")
		if _, err := client.Comment(ctx, event, plugins.FormatSimpleErrorResponse(event.GetAuthorName(), "I was unable to resume the test.\n Please try again later.")); err != nil {
			logger.Error(err, "unable to comment to github")
		}
	}

	if _, err := client.Comment(ctx, event, plugins.FormatSimpleResponse(event.GetAuthorName(), fmt.Sprintf("I resumed the testrun %s", run.Testrun.GetName()))); err != nil {
		logger.Error(err, "unable to comment to github")
	}
	return nil
}
