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

package tests

import (
	"context"
	"fmt"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/pkg/errors"
	"time"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
)

func (r *Runs) CreateTestrun(log logr.Logger, ctx context.Context, ghClient github.Client, event *github.GenericRequestEvent, tr *tmv1beta1.Testrun) (*tmv1beta1.Testrun, *StatusUpdater, error) {
	if runs.IsRunning(event) {
		return nil, nil, errors.New("A test is already running for this PR.")
	}

	if err := r.watch.Client().Create(ctx, tr); err != nil {
		return nil, nil, pluginerr.New("unable to create Testrun", err.Error())
	}
	log.Info(fmt.Sprintf("Testrun %s deployed", tr.Name))

	statusUpdater := NewStatusUpdater(log, ghClient, event)

	if err := statusUpdater.Init(ctx, tr); err != nil {
		log.Error(err, "unable to create comment", "Testrun", tr.Name)
	}

	return tr, statusUpdater, nil
}

func (r *Runs) Watch(log logr.Logger, ctx context.Context, statusUpdater *StatusUpdater, event *github.GenericRequestEvent, tr *tmv1beta1.Testrun, pollInterval, maxWaitTime time.Duration) (*tmv1beta1.Testrun, error) {
	if err := runs.Add(event, tr); err != nil {
		return nil, err
	}
	defer runs.Remove(event)

	dashboardURL, err := testrunner.GetTmDashboardURLForTestrun(r.watch.Client(), tr)
	if err != nil {
		log.V(3).Info("unable to get TestMachinery Dashboard URL", "error", err.Error())
	}

	testrunPhase := tmv1beta1.PhaseStatusInit
	err = r.watch.WatchUntil(maxWaitTime, tr.GetNamespace(), tr.GetName(), func(new *tmv1beta1.Testrun) (bool, error) {
		*tr = *new
		if tr.Status.State != "" {
			testrunPhase = tr.Status.Phase
			log.V(3).Info(fmt.Sprintf("Testrun %s is in %s phase. State: %s", tr.Name, testrunPhase, tr.Status.State))
		} else {
			log.V(3).Info(fmt.Sprintf("Testrun %s is in %s phase. Waiting ...", tr.Name, testrunPhase))
		}

		if testrunPhase != tmv1beta1.PhaseStatusInit {
			if err := statusUpdater.Update(ctx, tr, dashboardURL); err != nil {
				log.Error(err, "unable to update comment", "Testrun", tr.Name)
			}
		}

		return util.Completed(testrunPhase), nil
	})
	if err != nil {
		if err := statusUpdater.UpdateStatus(ctx, github.StateFailure, "timeout"); err != nil {
			log.Error(err, "unable to update comment", "Testrun", tr.Name)
		}
		return nil, pluginerr.New(fmt.Sprintf("maximum wait time of %s is exceeded", maxWaitTime.String()), "")
	}
	return tr, nil
}
