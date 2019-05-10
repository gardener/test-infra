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

package testflow

import (
	"fmt"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/argo"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

// NewNode creates a new TestflowNode for the internal DAG
func NewNode(parents []*Node, lastSerialNode, rootNode *Node, td *testdefinition.TestDefinition, step *Step, flow FlowIdentifier) *Node {
	td.SetPosition(string(flow), step.Row, step.Column)
	node := &Node{Parents: parents, TestDefinition: td, step: step.Info}

	for _, parent := range parents {
		parent.AddChild(node)
	}

	if lastSerialNode != nil {
		node.addTask(step.Info.ContinueOnError)
	}

	node.Status = &tmv1beta1.TestflowStepStatus{
		Phase: tmv1beta1.PhaseStatusInit,
		TestDefinition: tmv1beta1.TestflowStepStatusTestDefinition{
			Name:                  td.Info.Metadata.Name,
			Owner:                 td.Info.Spec.Owner,
			RecipientsOnFailure:   td.Info.Spec.RecipientsOnFailure,
			ActiveDeadlineSeconds: td.Info.Spec.ActiveDeadlineSeconds,
			Position:              td.GetPosition(),
		},
	}
	if td.Location != nil {
		node.Status.TestDefinition.Location = *td.Location.GetLocation()
	}

	return node
}

// AddChild adds a TestflowNode to the children
func (n *Node) AddChild(child *Node) {
	n.Children = append(n.Children, child)
}

// GetParentNames returns the names of a parent nodes
func (n *Node) GetParentNames() []string {
	names := []string{}
	for _, parent := range n.Parents {
		names = append(names, parent.Name())
	}
	return names
}

// Name return the unique name of the node's task
func (n *Node) Name() string {
	return n.Task.Name
}

// addTask adds an argo task to the node.AddTask
// Standard artifacts like kubeconfigs and the cloned repository are added as well as a execution condition if specified.
func (n *Node) addTask(continueOnError bool) {
	artifacts := []argov1.Artifact{
		{
			Name: "kubeconfigs",
			From: "{{workflow.outputs.artifacts.kubeconfigs}}",
		},
		{
			Name: "sharedFolder",
			From: "{{workflow.outputs.artifacts.sharedFolder}}",
		},
	}

	if n.TestDefinition.Location.Type() != tmv1beta1.LocationTypeLocal {
		artifacts = append(artifacts, argov1.Artifact{
			Name: "repo",
			From: fmt.Sprintf("{{workflow.outputs.artifacts.%s}}", n.TestDefinition.Location.Name()),
		})
	}
	task := argo.CreateTask(n.TestDefinition.Template.Name, n.TestDefinition.Template.Name, testmachinery.PHASE_RUNNING, continueOnError, n.GetParentNames(), artifacts)

	switch n.step.Condition {
	case tmv1beta1.ConditionTypeSuccess:
		task.When = fmt.Sprintf("{{workflow.status}} == Succeeded")
	case tmv1beta1.ConditionTypeError:
		task.When = fmt.Sprintf("{{workflow.status}} != Succeeded")
	}

	n.Task = task
}
