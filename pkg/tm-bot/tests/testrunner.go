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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
)

func CreateTestrun(log logr.Logger, ctx context.Context, k8sClient kubernetes.Interface, ghClient github.Client, event *github.GenericRequestEvent, tr *tmv1beta1.Testrun) (*tmv1beta1.Testrun, *StatusUpdater, error) {
	if err := Add(event, *tr); err != nil {
		return nil, nil, err
	}
	defer Remove(event)

	if err := k8sClient.Client().Create(ctx, tr); err != nil {
		return nil, nil, pluginerr.New("unable to create testrun", err.Error())
	}
	log.Info(fmt.Sprintf("Testrun %s deployed", tr.Name))

	statusUpdater := NewStatusUpdater(log, ghClient, event)

	if err := statusUpdater.Init(tr); err != nil {
		log.Error(err, "unable to create comment", "testrun", tr.Name)
	}

	return tr, statusUpdater, nil
}

func Watch(log logr.Logger, ctx context.Context, k8sClient kubernetes.Interface, statusUpdater *StatusUpdater, tr *tmv1beta1.Testrun, pollInterval, maxWaitTime time.Duration) (*tmv1beta1.Testrun, error) {
	argoUrl, err := testrunner.GetArgoURL(k8sClient, tr)
	if err != nil {
		log.WithValues("testrun", tr.Name).Error(err, "unable to construct argourl")
	}

	testrunPhase := tmv1beta1.PhaseStatusInit
	err = wait.PollImmediate(pollInterval, maxWaitTime, func() (bool, error) {
		testrun := &tmv1beta1.Testrun{}
		err := k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, testrun)
		if err != nil {
			log.Error(err, "cannot get testrun")
			return false, nil
		}
		tr = testrun

		if tr.Status.State != "" {
			testrunPhase = tr.Status.Phase
			log.V(3).Info(fmt.Sprintf("Testrun %s is in %s phase. State: %s", tr.Name, testrunPhase, tr.Status.State))
		} else {
			log.V(3).Info(fmt.Sprintf("Testrun %s is in %s phase. Waiting ...", tr.Name, testrunPhase))
		}

		if testrunPhase != tmv1beta1.PhaseStatusInit {
			if err := statusUpdater.Update(tr, argoUrl); err != nil {
				log.Error(err, "unable to update comment", "testrun", tr.Name)
			}
		}

		return util.Completed(testrunPhase), nil
	})
	if err != nil {
		if err := statusUpdater.UpdateStatus(github.StateFailure, "timeout"); err != nil {
			log.Error(err, "unable to update comment", "testrun", tr.Name)
		}
		return nil, pluginerr.New(fmt.Sprintf("maximum wait time of %s is exceeded", maxWaitTime.String()), "")
	}
	return tr, nil
}
