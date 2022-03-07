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
