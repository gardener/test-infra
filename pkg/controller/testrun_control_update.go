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
	"fmt"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery/argo"
	"k8s.io/apimachinery/pkg/types"
	"strings"
	"time"

	"github.com/gardener/test-infra/pkg/testmachinery/garbagecollection"
	"github.com/gardener/test-infra/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/test-infra/pkg/testmachinery"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ErrDeadlineExceeded indicates the operation exceeded its deadline for execution
// TODO: This needs to stay in sync with https://github.com/kubernetes/kubernetes/blob/7f23a743e8c23ac6489340bbb34fa6f1d392db9d/pkg/kubelet/active_deadline.go
// Needs to maintained on our own for now until message is exposed.
var ErrDeadlineExceeded = "Pod was active on the node longer than the specified deadline"

// handleActions handles any changes that trigger actions on a running workflow like annotations to resume a workflow
func (r *TestrunReconciler) handleActions(ctx *reconcileContext) error {
	if b, ok := ctx.tr.Annotations[common.ResumeTestrunAnnotation]; ok && b == "true" {
		// resume workflow
		argo.ResumeWorkflow(ctx.wf)
		if err := r.Client.Update(ctx.ctx, ctx.wf); err != nil {
			return err
		}

		delete(ctx.tr.Annotations, common.ResumeTestrunAnnotation)
		ctx.updated = true
	}
	return nil
}

func (r *TestrunReconciler) updateStatus(ctx *reconcileContext) (reconcile.Result, error) {
	log := r.Logger.WithValues("testrun", types.NamespacedName{Name: ctx.tr.Name, Namespace: ctx.tr.Namespace})

	if !ctx.tr.Status.StartTime.Equal(&ctx.wf.Status.StartedAt) {
		ctx.tr.Status.StartTime = &ctx.wf.Status.StartedAt
		ctx.updated = true
	}
	if ctx.tr.Status.Phase == "" {
		ctx.tr.Status.Phase = tmv1beta1.PhaseStatusPending
		ctx.updated = true
	}
	if !ctx.wf.Status.Completed() {
		updateStepsStatuses(ctx.tr, ctx.wf)
		ctx.updated = true
	}
	if ctx.wf.Status.Completed() {

		err := r.completeTestrun(ctx.tr, ctx.wf)
		if err != nil {
			return reconcile.Result{}, nil
		}
		ctx.updated = true
		log.Info("testrun completed")
	}

	if ctx.updated {
		err := r.Update(ctx.ctx, ctx.tr)
		if err != nil {
			log.Error(err, "unable to update testrun status")
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *TestrunReconciler) completeTestrun(tr *tmv1beta1.Testrun, wf *argov1.Workflow) error {
	log := r.Logger.WithValues("testrun", types.NamespacedName{Name: tr.Name, Namespace: tr.Namespace})
	log.Info("start collecting node status of Testrun")

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
	if err := garbagecollection.CleanWorkflowPods(r.Client, wf); err != nil {
		log.Error(err, "error while trying to cleanup pods")
	}

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
