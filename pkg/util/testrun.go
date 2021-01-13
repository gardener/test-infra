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
	"context"
	"fmt"
	argov1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
)

// TestrunStatusPhase determines the real testrun phase of a testrun by ignoring exit handler failures and system component failures if all other tests passed.
func TestrunStatusPhase(tr *tmv1beta1.Testrun) argov1alpha1.NodePhase {
	if tr.Status.Phase == tmv1beta1.PhaseStatusSuccess {
		return tmv1beta1.PhaseStatusSuccess
	}

	stepsRun := false
	for _, step := range tr.Status.Steps {
		if !stepsRun && (step.Phase != tmv1beta1.PhaseStatusInit && step.Phase != tmv1beta1.PhaseStatusSkipped) {
			stepsRun = true
		}
		if step.Phase == tmv1beta1.PhaseStatusInit || step.Phase == tmv1beta1.PhaseStatusSkipped {
			continue
		}
		if step.Phase != tmv1beta1.PhaseStatusSuccess && step.Annotations[common.AnnotationSystemStep] != "true" {
			return step.Phase
		}
	}

	if stepsRun {
		return tmv1beta1.PhaseStatusSuccess
	}

	return tr.Status.Phase
}

func IsSystemStep(step *tmv1beta1.StepStatus) bool {
	if len(step.Annotations) == 0 {
		return false
	}
	if _, ok := step.Annotations[common.AnnotationSystemStep]; ok {
		return true
	}
	return false
}

// Resume testruns resumes a testrun by adding the appropriate annotation to it
func ResumeTestrun(ctx context.Context, k8sClient client.Client, tr *tmv1beta1.Testrun) error {
	obj, err := client.ObjectKeyFromObject(tr)
	if err != nil {
		return err
	}
	if err := k8sClient.Get(ctx, obj, tr); err != nil {
		return err
	}
	if tr.Annotations == nil {
		tr.Annotations = make(map[string]string)
	}
	tr.Annotations[common.AnnotationResumeTestrun] = "true"
	if err := k8sClient.Update(ctx, tr); err != nil {
		return err
	}

	return nil
}

// TestrunProgress returns the progress of a testrun
func TestrunProgress(tr *tmv1beta1.Testrun) string {
	allSteps := 0
	completedSteps := 0
	for _, step := range tr.Status.Steps {
		if step.Annotations[common.AnnotationSystemStep] != "true" {
			allSteps++
			if step.Phase != tmv1beta1.PhaseStatusSkipped && Completed(step.Phase) {
				completedSteps++
			}
		}
	}

	return fmt.Sprintf("%d/%d", completedSteps, allSteps)
}

// OrderStepsStatus orders a list of step status of a testrun
func OrderStepsStatus(steps []*tmv1beta1.StepStatus) {
	sort.Sort(stepStatusList(steps))
}

type stepStatusList []*tmv1beta1.StepStatus

func (l stepStatusList) Len() int      { return len(l) }
func (l stepStatusList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l stepStatusList) Less(i, j int) bool {
	if l[i].StartTime == nil || l[j].StartTime == nil {
		return true
	}
	return l[j].StartTime.Before(l[i].StartTime)
}
