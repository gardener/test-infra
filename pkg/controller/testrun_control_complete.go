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

	"github.com/gardener/test-infra/pkg/testmachinery"

	log "github.com/sirupsen/logrus"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testrun"
)

func (r *TestrunReconciler) completeTestrun(ctx context.Context, tr *tmv1beta1.Testrun, wf *argov1.Workflow) error {
	log.Info("Collecting node status")

	tr.Status.Phase = wf.Status.Phase
	tr.Status.CompletionTime = &wf.Status.FinishedAt
	d := tr.Status.CompletionTime.Sub(tr.Status.StartTime.Time)
	tr.Status.Duration = int64(d.Seconds())

	parsedTr, err := testrun.New(tr)
	if err != nil {
		return fmt.Errorf("Error parsing testrun: %s", err.Error())
	}

	testflowStatus := [][]tmv1beta1.TestflowStepStatus{}

	root := parsedTr.Testflow.Flow.TestFlowRoot

	currentNode := root
	for len(currentNode.Children) > 0 {
		stepStatus := []tmv1beta1.TestflowStepStatus{}
		for _, node := range currentNode.Children {
			log.Debugf("Collecting status of node %s", node.Task.Name)
			currentNode = node
			argoNodeStatus := getArgoNodeStatus(wf, currentNode.TestDefinition.Template.Metadata.Annotations)

			if argoNodeStatus == nil {
				stepStatus = append(stepStatus, tmv1beta1.TestflowStepStatus{
					Phase: argov1.NodeSkipped,
					TestDefinition: tmv1beta1.TestflowStepStatusTestDefinition{
						Name:     node.TestDefinition.Info.Metadata.Name,
						Location: *node.TestDefinition.Location.GetLocation(),
					},
				})
				continue
			}

			status := *argoNodeStatus
			d := status.FinishedAt.Sub(argoNodeStatus.StartedAt.Time)
			stepStatus = append(stepStatus, tmv1beta1.TestflowStepStatus{
				Phase:             status.Phase,
				StartTime:         &status.StartedAt,
				CompletionTime:    &status.FinishedAt,
				Duration:          int64(d.Seconds()),
				ExportArtifactKey: getNodeExportKey(status.Outputs),
				TestDefinition: tmv1beta1.TestflowStepStatusTestDefinition{
					Name:                node.TestDefinition.Info.Metadata.Name,
					Location:            *node.TestDefinition.Location.GetLocation(),
					Owner:               node.TestDefinition.Info.Spec.Owner,
					RecipientsOnFailure: node.TestDefinition.Info.Spec.RecipientsOnFailure,
				},
			})
		}

		testflowStatus = append(testflowStatus, stepStatus)
	}

	tr.Status.Steps = testflowStatus

	return nil
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
