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

package resume

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

type resume struct {
	runID     string
	log       logr.Logger
	k8sClient kubernetes.Interface
}

func New(log logr.Logger, k8sClient kubernetes.Interface) plugins.Plugin {
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
	return github.AuthorizationOrg
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
		if _, err := client.Comment(event, plugins.FormatSimpleErrorResponse(event.GetAuthorName(), "There are no running tests for this PR")); err != nil {
			r.log.Error(err, "unable to comment to github")
		}
	}
	logger := r.log.WithValues("testrun", run.Testrun.GetName(), "namespace", run.Testrun.GetNamespace())

	tr := &v1beta1.Testrun{}
	if err := r.k8sClient.Client().Get(ctx, client2.ObjectKey{Name: run.Testrun.Name, Namespace: run.Testrun.Namespace}, tr); err != nil {
		logger.Error(err, "unable to to fetch testrun")
		if _, err := client.Comment(event, plugins.FormatSimpleErrorResponse(event.GetAuthorName(), "There are no running tests for this PR")); err != nil {
			logger.Error(err, "unable to comment to github")
		}
	}

	if err := util.ResumeTestrun(ctx, r.k8sClient, tr); err != nil {
		logger.Error(err, "unable to resume testrun")
		if _, err := client.Comment(event, plugins.FormatSimpleErrorResponse(event.GetAuthorName(), "I was unable to resume the test.\n Please try again later.")); err != nil {
			logger.Error(err, "unable to comment to github")
		}
	}

	if _, err := client.Comment(event, plugins.FormatSimpleResponse(event.GetAuthorName(), fmt.Sprintf("I resumed the testrun %s", run.Testrun.GetName()))); err != nil {
		logger.Error(err, "unable to comment to github")
	}
	return nil
}
