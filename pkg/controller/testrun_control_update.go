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
	"reflect"

	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	"github.com/gardener/test-infra/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/test-infra/pkg/testmachinery"

	log "github.com/sirupsen/logrus"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

func (r *TestrunReconciler) updateStatus(ctx context.Context, tr *tmv1beta1.Testrun, wf *argov1.Workflow) (reconcile.Result, error) {
	if !tr.Status.StartTime.Equal(&wf.Status.StartedAt) {
		tr.Status.StartTime = &wf.Status.StartedAt
	}
	if tr.Status.Phase == "" {
		tr.Status.Phase = tmv1beta1.PhaseStatusPending
	}
	if !wf.Status.Completed() {
		updateStepsStatus(tr, wf)
	}
	if wf.Status.Completed() {

		err := completeTestrun(tr, wf)
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

func completeTestrun(tr *tmv1beta1.Testrun, wf *argov1.Workflow) error {
	log.Info("Collecting node status")

	tr.Status.Phase = wf.Status.Phase
	tr.Status.CompletionTime = &wf.Status.FinishedAt
	trDuration := tr.Status.CompletionTime.Sub(tr.Status.StartTime.Time)
	tr.Status.Duration = int64(trDuration.Seconds())

	updateStepsStatus(tr, wf)

	// Set all init steps to skipped if testrun is completed.
	for _, steps := range tr.Status.Steps {
		for _, stepStatus := range steps {
			if stepStatus.Phase == tmv1beta1.PhaseStatusInit {
				stepStatus.Phase = argov1.NodeSkipped
			}
		}
	}

	return nil
}

func updateStepsStatus(tr *tmv1beta1.Testrun, wf *argov1.Workflow) {
	completedSteps := 0
	numSteps := 0
	for row, steps := range tr.Status.Steps {
		for column, stepStatus := range steps {
			numSteps++

			argoNodeStatus := getArgoNodeStatus(wf, testdefinition.GetAnnotations(stepStatus.TestDefinition.Name, string(testflow.FlowIDTest), row, column))
			if argoNodeStatus == nil {
				continue
			}

			stepDuration := argoNodeStatus.FinishedAt.Sub(argoNodeStatus.StartedAt.Time)
			stepStatus.Phase = argoNodeStatus.Phase
			stepStatus.StartTime = &argoNodeStatus.StartedAt
			stepStatus.CompletionTime = &argoNodeStatus.FinishedAt
			stepStatus.Duration = int64(stepDuration.Seconds())
			stepStatus.ExportArtifactKey = getNodeExportKey(argoNodeStatus.Outputs)
			if util.Completed(stepStatus.Phase) {
				completedSteps++
			}
		}
	}

	tr.Status.State = fmt.Sprintf("Testmachinery executed %d/%d Steps", completedSteps, numSteps)
}

func getArgoNodeStatus(wf *argov1.Workflow, annotations map[string]string) *argov1.NodeStatus {
	for _, node := range wf.Status.Nodes {
		if nodeIsAtPosition(wf, node.TemplateName, annotations) {
			return &node
		}
	}

	return nil
}

// nodeIsAtPosition checks if the wf node status is at the at the current position(raw, column)
// This is archieved by getting the node's corrresponding template and check the templates annotations.
func nodeIsAtPosition(wf *argov1.Workflow, templateName string, annotations map[string]string) bool {
	for _, template := range wf.Spec.Templates {
		if template.Name == templateName && reflect.DeepEqual(template.Metadata.Annotations, annotations) {
			return true
		}
	}
	return false
}

func getNodeExportKey(outputs *argov1.Outputs) string {
	if outputs == nil {
		return ""
	}
	for _, artifact := range outputs.Artifacts {
		if artifact.Name == testmachinery.ExportArtifact {
			return artifact.ArtifactLocation.S3.Key
		}
	}
	return ""
}
