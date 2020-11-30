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

package testrunner

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/utils/retry"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	trerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"time"
)

// SetRunID sets the provided run id as label and adds it to the metadata
func (r *Run) SetRunID(id string) {
	if len(id) == 0 {
		return
	}
	if r.Testrun.Labels == nil {
		r.Testrun.Labels = make(map[string]string, 1)
	}
	r.Testrun.Labels[common.LabelTestrunExecutionGroup] = id
	if r.Metadata != nil {
		r.Metadata.Testrun.ExecutionGroup = id
	}
}

// SetTMDashboardURL sets the provided dashboard URL as annotation
func (r *Run) SetTMDashboardURL(url string) {
	if url == "" {
		return
	}
	if r.Testrun.Annotations == nil {
		r.Testrun.Annotations = make(map[string]string, 1)
	}
	r.Testrun.Annotations[common.AnnotationTMDashboardURL] = url
}

func (r *Run) Exec(log logr.Logger, config *Config, prefix string) {
	ctx := context.Background()
	defer ctx.Done()
	newTR := r.Testrun.DeepCopy()

	// Remove legacy name attribute. Instead enforce usage of generateName.
	newTR.Name = ""
	newTR.GenerateName = prefix
	newTR.Namespace = config.Namespace

	err := retry.UntilTimeout(ctx, 30*time.Second, 5*time.Minute, func(ctx context.Context) (bool, error) {
		err := config.Watch.Client().Create(ctx, newTR)
		if err != nil {
			log.Error(err, "unable to create testrun. Retrying...")
			return retry.MinorError(err)
		}
		return retry.Ok()
	})
	if err != nil {
		r.Error = trerrors.NewNotCreatedError(fmt.Sprintf("cannot create testrun: %s", err.Error()))
		return
	}

	*r.Testrun = *newTR
	r.Metadata.Testrun.ID = newTR.GetName()
	log.Info(fmt.Sprintf("Testrun %s deployed", newTR.Name))

	if TMDashboardHost, err := GetTMDashboardHost(config.Watch.Client()); err == nil {
		log.Info(fmt.Sprintf("TestMachinery Dashboard for Testrun %s: %s", r.Testrun.Name, GetTmDashboardURLFromHostForTestrun(TMDashboardHost, r.Testrun)))
	}
	if argoUrl, err := GetArgoURL(ctx, config.Watch.Client(), r.Testrun); err == nil {
		log.WithValues("testrun", r.Testrun.GetName()).Info(fmt.Sprintf("Argo workflow: %s", argoUrl))
	}

	testrunPhase := tmv1beta1.PhaseStatusInit
	err = config.Watch.WatchUntil(config.Timeout, r.Testrun.GetNamespace(), r.Testrun.GetName(), func(new *tmv1beta1.Testrun) (bool, error) {
		*r.Testrun = *new
		if r.Testrun.Status.State != "" {
			testrunPhase = r.Testrun.Status.Phase
			log.Info(fmt.Sprintf("Testrun %s is in %s phase. State: %s", r.Testrun.GetName(), testrunPhase, r.Testrun.Status.State))
		} else {
			log.Info(fmt.Sprintf("Testrun %s is in %s phase. Waiting ...", r.Testrun.GetName(), testrunPhase))
		}
		return util.Completed(testrunPhase), nil
	})
	if err != nil {
		r.Testrun.Status.Phase = tmv1beta1.PhaseStatusTimeout
		r.Error = trerrors.NewTimeoutError(fmt.Sprintf("maximum wait time of %d is exceeded by Testrun %s", config.Timeout, r.Testrun.GetName()))
	}

	fmt.Println(RunList{r}.RenderTable())
}
