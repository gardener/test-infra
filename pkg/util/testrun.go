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

package util

import (
	argov1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
)

// TestrunStatusPhase determines the real testrun phase of a testrun by ignoring exit handler failures and system component failures if all other tests passed.
func TestrunStatusPhase(tr *v1beta1.Testrun) argov1alpha1.NodePhase {
	if tr.Status.Phase == v1beta1.PhaseStatusSuccess {
		return v1beta1.PhaseStatusSuccess
	}

	for _, step := range tr.Status.Steps {
		if step.Phase == v1beta1.PhaseStatusInit {
			continue
		}
		if step.Phase != v1beta1.PhaseStatusSuccess && step.Annotations[common.AnnotationSystemStep] != "true" {
			return step.Phase
		}
	}

	return v1beta1.PhaseStatusSuccess
}
