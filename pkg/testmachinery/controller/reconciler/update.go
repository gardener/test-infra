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

package reconciler

import (
	"context"
	"fmt"
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
func (r *TestmachineryReconciler) handleActions(ctx context.Context, rCtx *reconcileContext) error {
	return r.resumeAction(ctx, rCtx)
}

func (r *TestmachineryReconciler) updateStatus(ctx context.Context, rCtx *reconcileContext) (reconcile.Result, error) {
	log := r.Logger.WithValues("testrun", types.NamespacedName{Name: rCtx.tr.Name, Namespace: rCtx.tr.Namespace})

	if !rCtx.tr.Status.StartTime.Equal(&rCtx.wf.Status.StartedAt) {
		rCtx.tr.Status.StartTime = &rCtx.wf.Status.StartedAt
		rCtx.updated = true
	}
	if rCtx.tr.Status.Phase == "" {
		rCtx.tr.Status.Phase = tmv1beta1.PhaseStatusPending
		rCtx.updated = true
	}
	if !rCtx.wf.Status.Completed() {
		r.updateStepsStatus(rCtx)
		rCtx.updated = true
	} else {
		err := r.completeTestrun(rCtx)
		if err != nil {
			return reconcile.Result{}, nil
		}
		rCtx.updated = true
		log.Info("testrun completed")
	}

	if rCtx.updated {
		rCtx.tr.Status.ObservedGeneration = rCtx.tr.Generation
		if err := r.Status().Update(ctx, rCtx.tr); err != nil {
			log.Error(err, "unable to update testrun status")
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *TestmachineryReconciler) completeTestrun(rCtx *reconcileContext) error {
	log := r.Logger.WithValues("testrun", types.NamespacedName{Name: rCtx.tr.Name, Namespace: rCtx.tr.Namespace})
	log.Info("start collecting node status of Testrun")

	rCtx.tr.Status.Phase = rCtx.wf.Status.Phase
	rCtx.tr.Status.CompletionTime = &rCtx.wf.Status.FinishedAt
	trDuration := rCtx.tr.Status.CompletionTime.Sub(rCtx.tr.Status.StartTime.Time)
	rCtx.tr.Status.Duration = int64(trDuration.Seconds())

	r.updateStepsStatus(rCtx)

	// Set all init steps to skipped if testrun is completed.
	for _, step := range rCtx.tr.Status.Steps {
		if step.Phase == tmv1beta1.PhaseStatusInit {
			step.Phase = argov1.NodeSkipped
		}
	}

	// collect results
	metadata, err := r.collector.GetMetadata(rCtx.tr)
	if err != nil {
		return err
	}
	if err := r.collector.Collect(rCtx.tr, metadata); err != nil {
		return err
	}

	// cleanup pods to remove workload from the api server
	// logs are still accessible through "archiveLogs" option in argo
	if err := garbagecollection.CleanWorkflowPods(r, rCtx.wf); err != nil {
		log.Error(err, "error while trying to cleanup pods")
	}

	return nil
}

func (r *TestmachineryReconciler) updateStepsStatus(rCtx *reconcileContext) {
	r.Logger.V(3).Info("update step status")
	completedSteps := 0
	numSteps := len(rCtx.tr.Status.Steps)
	stepSpecs := getStepsSpecs(rCtx.tr)

	for _, step := range rCtx.tr.Status.Steps {
		if util.Completed(step.Phase) {
			completedSteps++
			continue
		}
		r.Logger.V(5).Info("update status", "step", step.Name)

		r.checkResume(rCtx, stepSpecs[step.Position.Step], step.Name)

		argoNodeStatus := getNodeStatusByName(rCtx.wf, step.Name)
		// continue with the next status if no corresponding argo status can be found yet.
		if argoNodeStatus == nil {
			continue
		}

		if strings.Contains(argoNodeStatus.Message, ErrDeadlineExceeded) {
			r.Logger.V(5).Info("update timeout step status", "step", step.Name)
			if step.StartTime == nil {
				step.StartTime = &argoNodeStatus.StartedAt
			}

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

	rCtx.tr.Status.State = fmt.Sprintf("Testmachinery executed %d/%d Steps", completedSteps, numSteps)
}

func getStepsSpecs(tr *tmv1beta1.Testrun) map[string]*tmv1beta1.DAGStep {
	steps := make(map[string]*tmv1beta1.DAGStep, len(tr.Spec.TestFlow))
	for _, step := range tr.Spec.TestFlow {
		steps[step.Name] = step
	}
	return steps
}

func getNodeStatusByName(wf *argov1.Workflow, templateName string) *argov1.NodeStatus {
	for _, nodeStatus := range wf.Status.Nodes {
		// need to check the prefix as argo appends the name of the testflow to the nodes name: <templateName>.<testflow-name>
		if nodeStatus.DisplayName == templateName {
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
