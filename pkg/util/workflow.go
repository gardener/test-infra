// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// WorkflowPhase calculates the phase for a completed workflow.
// In contrast to the argo status, we need to also consider continueOn steps as failures.
func WorkflowPhase(wf *argov1.Workflow) argov1.WorkflowPhase {
	if !wf.Status.Phase.Completed() {
		return wf.Status.Phase
	}

	phase := wf.Status.Phase
	for _, node := range wf.Status.Nodes {
		if node.Phase == tmv1beta1.StepPhaseError {
			return tmv1beta1.RunPhaseError
		}
		if node.Phase == tmv1beta1.StepPhaseFailed {
			return tmv1beta1.RunPhaseFailed
		}
	}

	return phase
}
