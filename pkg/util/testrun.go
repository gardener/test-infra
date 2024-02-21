// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"fmt"
	"sort"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
)

// TestrunStatusPhase determines the real testrun phase of a testrun by ignoring exit handler failures and system component failures if all other tests passed.
func TestrunStatusPhase(tr *tmv1beta1.Testrun) argov1alpha1.WorkflowPhase {
	if tr.Status.Phase == tmv1beta1.RunPhaseSuccess {
		return tmv1beta1.RunPhaseSuccess
	}

	stepsRun := false
	for _, step := range tr.Status.Steps {
		if !stepsRun && (step.Phase != tmv1beta1.StepPhaseInit && step.Phase != tmv1beta1.StepPhaseSkipped) {
			stepsRun = true // at least "some" steps were run
		}
		if step.Phase == tmv1beta1.StepPhaseInit || step.Phase == tmv1beta1.StepPhaseSkipped {
			continue
		}
		if step.Phase != tmv1beta1.StepPhaseSuccess && step.Annotations[common.AnnotationSystemStep] != "true" {
			switch step.Phase {
			case tmv1beta1.StepPhasePending:
				return tmv1beta1.RunPhasePending
			case tmv1beta1.StepPhaseFailed:
				return tmv1beta1.RunPhaseFailed
			case tmv1beta1.StepPhaseError:
				return tmv1beta1.RunPhaseError
			case tmv1beta1.StepPhaseRunning:
				return tmv1beta1.RunPhaseRunning
			case tmv1beta1.StepPhaseTimeout:
				return tmv1beta1.RunPhaseTimeout
			}
		}
	}

	if stepsRun {
		return tmv1beta1.RunPhaseSuccess
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
	key := client.ObjectKeyFromObject(tr)
	if err := k8sClient.Get(ctx, key, tr); err != nil {
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
			if step.Phase != tmv1beta1.StepPhaseSkipped && CompletedStep(step.Phase) {
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
