// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"
	"strings"
	"time"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/garbagecollection"
	"github.com/gardener/test-infra/pkg/util"
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
		rCtx.tr.Status.Phase = tmv1beta1.RunPhasePending
		rCtx.updated = true
	}
	if !rCtx.wf.Status.Phase.Completed() {
		r.updateStepsStatus(rCtx)
		rCtx.updated = true
	} else {
		err := r.completeTestrun(rCtx)
		if err != nil {
			return reconcile.Result{}, err
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

	rCtx.tr.Status.Phase = util.WorkflowPhase(rCtx.wf)
	rCtx.tr.Status.CompletionTime = &rCtx.wf.Status.FinishedAt
	trDuration := rCtx.tr.Status.CompletionTime.Sub(rCtx.tr.Status.StartTime.Time)
	rCtx.tr.Status.Duration = int64(trDuration.Seconds())

	r.updateStepsStatus(rCtx)

	// Set all init steps to skipped if testrun is completed.
	for _, step := range rCtx.tr.Status.Steps {
		if step.Phase == tmv1beta1.StepPhaseInit {
			step.Phase = argov1.NodeSkipped
		}
	}

	// collect results
	if r.collector != nil {
		metadata, err := r.collector.GetMetadata(rCtx.tr)
		if err != nil {
			return err
		}
		if err := r.collector.Collect(rCtx.tr, metadata); err != nil {
			return err
		}
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
		if util.CompletedStep(step.Phase) {
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

			testDuration := time.Duration(step.TestDefinition.ActiveDeadlineSeconds.IntValue()) * time.Second
			completionTime := metav1.NewTime(step.StartTime.Add(testDuration))

			step.Phase = tmv1beta1.StepPhaseTimeout
			step.Duration = int64(step.TestDefinition.ActiveDeadlineSeconds.IntValue())
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

		if util.CompletedStep(step.Phase) {
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
		if artifact.Name == testmachinery.ExportArtifact && artifact.S3 != nil {
			return artifact.S3.Key
		}
	}
	return ""
}
