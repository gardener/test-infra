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

package reconciler

import (
	"context"
	"fmt"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/argo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// Resume a workflow if a specific annotation is set
func (r *TestmachineryReconciler) resumeAction(ctx context.Context, rCtx *reconcileContext) error {
	if b, ok := rCtx.tr.Annotations[common.AnnotationResumeTestrun]; ok && b == "true" {
		// resume workflow
		argo.ResumeWorkflow(rCtx.wf)
		if err := r.Client.Update(ctx, rCtx.wf); err != nil {
			return err
		}

		delete(rCtx.tr.Annotations, common.AnnotationResumeTestrun)
		rCtx.updated = true
	}
	return nil
}

// checkResume checks if a step is a paused step and creates an timer to resume after a dedicated time
func (r *TestmachineryReconciler) checkResume(rCtx *reconcileContext, step *v1beta1.DAGStep, stepStatusName string) {
	// check if a pause step exists
	argoNodeStatus := getNodeStatusByName(rCtx.wf, testmachinery.GetPauseTaskName(stepStatusName))
	if argoNodeStatus == nil {
		return
	}

	// node has to be in running state
	if argoNodeStatus.Phase != v1beta1.PhaseStatusRunning {
		return
	}

	timeoutSeconds := common.DefaultPauseTimeout
	if step.Pause != nil && step.Pause.ResumeTimeoutSeconds != nil {
		timeoutSeconds = *step.Pause.ResumeTimeoutSeconds
	}
	timeout := time.Duration(timeoutSeconds) * time.Second

	if err := r.addTimer(resumeTimerKey(rCtx.tr), calculateTimer(timeout, argoNodeStatus.StartedAt), func() {
		ctx := context.Background()
		defer ctx.Done()
		err := retry.UntilTimeout(ctx, 20*time.Second, 5*time.Minute, func(ctx context.Context) (bool, error) {
			r.Logger.V(5).Info("resuming workflow", "name", rCtx.tr.Status.Workflow, "namespace", rCtx.tr.GetNamespace())
			wf := &v1alpha1.Workflow{}
			if err := r.Client.Get(ctx, client.ObjectKey{Name: rCtx.tr.Status.Workflow, Namespace: rCtx.tr.GetNamespace()}, wf); err != nil {
				r.Logger.V(5).Info(err.Error())
				return retry.MinorError(err)
			}
			if ok := argo.ResumeWorkflowStep(wf, argoNodeStatus.Name); ok {
				if err := r.Client.Update(ctx, wf); err != nil {
					r.Logger.V(5).Info(err.Error())
					return retry.MinorError(err)
				}
			}
			return retry.Ok()
		})
		if err != nil {
			r.Logger.Error(err, "unable to resume workflow")
		}
	}); err != nil {
		r.Logger.Error(err, "unable to add resume timer")
	}
}

// resumeTimerKey generates a unique key for a resume timer of a testrun.
// A testrun is supposed to have only one timer.
func resumeTimerKey(testrun *v1beta1.Testrun) string {
	return fmt.Sprintf("%s/%s/resume", testrun.Namespace, testrun.Name)
}

// calculateTimer calculates the remaining time for a step
// if the time is elapsed a duration of zero is returned
func calculateTimer(pauseTimeout time.Duration, startTime metav1.Time) time.Duration {
	elapsedTime := time.Now().Sub(startTime.Time)
	remainingDuration := pauseTimeout - elapsedTime
	if remainingDuration <= 0 {
		return time.Duration(0)
	}
	return remainingDuration
}
