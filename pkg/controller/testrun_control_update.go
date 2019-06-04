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

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gardener/test-infra/pkg/testmachinery/garbagecollection"
	"github.com/gardener/test-infra/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/test-infra/pkg/testmachinery"

	log "github.com/sirupsen/logrus"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ErrDeadlineExceeded indicates the operation exceeded its deadline for execution
// TODO: This needs to stay in sync with https://github.com/kubernetes/kubernetes/blob/7f23a743e8c23ac6489340bbb34fa6f1d392db9d/pkg/kubelet/active_deadline.go
// Needs to maintained on our own for now until message is exposed.
var ErrDeadlineExceeded = "Pod was active on the node longer than the specified deadline"

func (r *TestrunReconciler) updateStatus(ctx context.Context, tr *tmv1beta1.Testrun, wf *argov1.Workflow) (reconcile.Result, error) {
	if !tr.Status.StartTime.Equal(&wf.Status.StartedAt) {
		tr.Status.StartTime = &wf.Status.StartedAt
	}
	if tr.Status.Phase == "" {
		tr.Status.Phase = tmv1beta1.PhaseStatusPending
	}
	if !wf.Status.Completed() {
		updateStepsStatuses(tr, wf)
	}
	if wf.Status.Completed() {

		err := r.completeTestrun(tr, wf)
		if err != nil {
			return reconcile.Result{}, nil
		}

		log.Infof("Testrun %s completed", tr.Name)
	}

	err := r.Update(ctx, tr)
	if err != nil {
		log.Errorf("Error updating Testrun status %s: %s", tr.Name, err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *TestrunReconciler) completeTestrun(tr *tmv1beta1.Testrun, wf *argov1.Workflow) error {
	log.Infof("Collecting node status of Testrun %s in Namespace %s", tr.Name, tr.Namespace)

	tr.Status.Phase = wf.Status.Phase
	tr.Status.CompletionTime = &wf.Status.FinishedAt
	trDuration := tr.Status.CompletionTime.Sub(tr.Status.StartTime.Time)
	tr.Status.Duration = int64(trDuration.Seconds())

	updateStepsStatuses(tr, wf)

	// Set all init steps to skipped if testrun is completed.
	for _, step := range tr.Status.Steps {
		if step.Phase == tmv1beta1.PhaseStatusInit {
			step.Phase = argov1.NodeSkipped
		}
	}

	// cleanup pods to remove workload from the api server
	// logs are still accessible through "archiveLogs" option in argo
	garbagecollection.CleanWorkflowPods(r.Client, wf)

	return nil
}

func updateStepsStatuses(tr *tmv1beta1.Testrun, wf *argov1.Workflow) {
	completedSteps := 0
	numSteps := len(tr.Status.Steps)

	for _, step := range tr.Status.Steps {
		if util.Completed(step.Phase) {
			completedSteps++
			continue
		}
		argoNodeStatus := getNodeStatusByName(wf, step.Name)
		// continue with the next status if no corresponding argo status can be found yet.
		if argoNodeStatus == nil {
			continue
		}

		if strings.Contains(argoNodeStatus.Message, ErrDeadlineExceeded) {
			testDuration := time.Duration(*step.TestDefinition.ActiveDeadlineSeconds) * time.Second
			completionTime := metav1.NewTime(step.StartTime.Add(testDuration))

			step.Phase = tmv1beta1.PhaseStatusTimeout
			step.Duration = *step.TestDefinition.ActiveDeadlineSeconds
			step.CompletionTime = &completionTime
			step.PodName = argoNodeStatus.ID

			completedSteps++
			continue
		}

		step.Phase = argoNodeStatus.Phase
		step.ExportArtifactKey = getNodeExportKey(argoNodeStatus.Outputs)
		step.PodName = argoNodeStatus.ID

		if !argoNodeStatus.StartedAt.IsZero() {
			step.StartTime = &argoNodeStatus.StartedAt
		}
		if !argoNodeStatus.FinishedAt.IsZero() {
			stepDuration := argoNodeStatus.FinishedAt.Sub(argoNodeStatus.StartedAt.Time)
			step.CompletionTime = &argoNodeStatus.FinishedAt
			step.Duration = int64(stepDuration.Seconds())
		}

		if util.Completed(step.Phase) {
			completedSteps++
		}
	}

	tr.Status.State = fmt.Sprintf("Testmachinery executed %d/%d Steps", completedSteps, numSteps)
}

func getNodeStatusByName(wf *argov1.Workflow, templateName string) *argov1.NodeStatus {
	for _, nodeStatus := range wf.Status.Nodes {
		if nodeStatus.TemplateName == templateName {
			return &nodeStatus
		}
	}

	return nil
}

func getNodeExportKey(outputs *argov1.Outputs) string {
	if outputs == nil {
		return ""
	}
	for _, artifact := range outputs.Artifacts {
		if artifact.Name == testmachinery.ExportArtifact && artifact.ArtifactLocation.S3 != nil {
			return artifact.ArtifactLocation.S3.Key
		}
	}
	return ""
}
