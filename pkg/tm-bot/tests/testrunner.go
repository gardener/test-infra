// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/gardener/test-infra/pkg/util"
)

func (r *Runs) CreateTestrun(ctx context.Context, log logr.Logger, statusUpdater *StatusUpdater, event *github.GenericRequestEvent, tr *tmv1beta1.Testrun) error {
	if runs.IsRunning(event) {
		return errors.New("A test is already running for this PR.")
	}

	if err := r.watch.Client().Create(ctx, tr); err != nil {
		return pluginerr.New("unable to create Testrun", err.Error())
	}
	log.Info(fmt.Sprintf("Testrun %s deployed", tr.Name))

	if err := statusUpdater.Init(ctx, tr); err != nil {
		log.Error(err, "unable to create comment", "Testrun", tr.Name)
	}

	return nil
}

func (r *Runs) Watch(ctx context.Context, log logr.Logger, statusUpdater *StatusUpdater, event *github.GenericRequestEvent, tr *tmv1beta1.Testrun, pollInterval, maxWaitTime time.Duration) (*tmv1beta1.Testrun, error) {
	if err := runs.Add(event, tr); err != nil {
		return nil, err
	}
	defer runs.Remove(event)

	dashboardURL, err := testrunner.GetTmDashboardURLForTestrun(r.watch.Client(), tr)
	if err != nil {
		log.V(3).Info("unable to get TestMachinery Dashboard URL", "error", err.Error())
	}

	testrunPhase := tmv1beta1.RunPhaseInit
	err = r.watch.WatchUntil(maxWaitTime, tr.GetNamespace(), tr.GetName(), func(new *tmv1beta1.Testrun) (bool, error) {
		*tr = *new
		if tr.Status.State != "" {
			testrunPhase = tr.Status.Phase
			log.V(3).Info(fmt.Sprintf("Testrun %s is in %s phase. State: %s", tr.Name, testrunPhase, tr.Status.State))
		} else {
			log.V(3).Info(fmt.Sprintf("Testrun %s is in %s phase. Waiting ...", tr.Name, testrunPhase))
		}

		if testrunPhase != tmv1beta1.RunPhaseInit {
			if err := statusUpdater.Update(ctx, tr, dashboardURL); err != nil {
				log.Error(err, "unable to update comment", "Testrun", tr.Name)
			}
		}

		return util.CompletedRun(testrunPhase), nil
	})
	if err != nil {
		if err := statusUpdater.UpdateStatus(ctx, github.StateFailure, "timeout"); err != nil {
			log.Error(err, "unable to update comment", "Testrun", tr.Name)
		}
		return nil, pluginerr.New(fmt.Sprintf("maximum wait time of %s is exceeded", maxWaitTime.String()), "")
	}
	return tr, nil
}
