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

package node

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/argo"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

// CreateNodesFromStep creates new nodes from a step and adds default configuration
func CreateNodesFromStep(step *tmv1beta1.DAGStep, loc locations.Locations, globalConfig []*config.Element, flowID string) ([]*Node, error) {
	testdefinitions, err := loc.GetTestDefinitions(step.Definition)
	if err != nil {
		return nil, err
	}

	nodes := make([]*Node, 0)
	for _, td := range testdefinitions {
		node := NewNode(td, step, flowID)
		if err := td.AddConfig(config.New(step.Definition.Config)); err != nil {
			return nil, err
		}
		if err := td.AddConfig(globalConfig); err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// NewNode creates a new TestflowNode for the internal DAG
func NewNode(td *testdefinition.TestDefinition, step *tmv1beta1.DAGStep, flow string) *Node {
	// create hash or unique name for testdefinition + step + flow
	name := GetUniqueName(td, step, flow)
	td.SetName(name)
	node := &Node{TestDefinition: td, step: step}

	node.Status = &tmv1beta1.StepStatus{
		Name:  name,
		Phase: tmv1beta1.PhaseStatusInit,
		TestDefinition: tmv1beta1.StepStatusTestDefinition{
			Name:                  td.Info.Metadata.Name,
			Owner:                 td.Info.Spec.Owner,
			RecipientsOnFailure:   td.Info.Spec.RecipientsOnFailure,
			ActiveDeadlineSeconds: td.Info.Spec.ActiveDeadlineSeconds,
			Position: map[string]string{
				"flow": flow,
			},
		},
	}
	if step != nil {
		node.Status.TestDefinition.Position["step"] = step.Name
	}
	if td.Location != nil {
		node.Status.TestDefinition.Location = *td.Location.GetLocation()
	}

	return node
}

// AddChildren adds Nodes as children
func (n *Node) AddChildren(children ...*Node) {
	n.Children = append(n.Children, children...)
}

// ClearParent removes a parent node from the current node
func (n *Node) RemoveChild(child *Node) {
	for i, val := range n.Children {
		if val == child {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
		}
	}
}

// ClearChildren removes all children form the current node
func (n *Node) ClearChildren() {
	n.Children = make(List, 0)
}

// AddParents adds Nodes as parents
func (n *Node) AddParents(parents ...*Node) {
	n.Parents = append(n.Parents, parents...)
}

// ClearParent removes a parent node from the current node
func (n *Node) RemoveParent(parent *Node) {
	for i, val := range n.Parents {
		if val == parent {
			n.Parents = append(n.Parents[:i], n.Parents[i+1:]...)
		}
	}
}

// ClearParents removes all parents from the current node
func (n *Node) ClearParents() {
	n.Parents = make(List, 0)
}

// ParentNames returns the names of a parent nodes
func (n *Node) ParentNames() []string {
	names := make([]string, 0)
	for _, parent := range n.Parents {
		names = append(names, parent.Name())
	}
	return names
}

// Name return the unique name of the node's task
func (n *Node) Name() string {
	return n.TestDefinition.Template.Name
}

func (n *Node) Task() argov1.DAGTask {
	artifacts := make([]argov1.Artifact, 0)
	// should only be nil if the node is the root node
	if n.inputSource != nil {
		artifacts = []argov1.Artifact{
			{
				Name: "kubeconfigs",
				From: fmt.Sprintf("{{tasks.%s.outputs.artifacts.kubeconfigs}}", n.inputSource.Name()),
			},
			{
				Name: "sharedFolder",
				From: fmt.Sprintf("{{tasks.%s.outputs.artifacts.sharedFolder}}", n.inputSource.Name()),
			},
		}

		if n.TestDefinition.Location != nil && n.TestDefinition.Location.Type() != tmv1beta1.LocationTypeLocal {
			artifacts = append(artifacts, argov1.Artifact{
				Name: "repo",
				From: fmt.Sprintf("{{workflow.outputs.artifacts.%s}}", n.TestDefinition.Location.Name()),
			})
		}
	}

	task := argo.CreateTask(n.TestDefinition.GetName(), n.TestDefinition.GetName(), testmachinery.PHASE_RUNNING, n.step.Definition.ContinueOnError, n.ParentNames(), artifacts)

	switch n.step.Definition.Condition {
	case tmv1beta1.ConditionTypeSuccess:
		task.When = fmt.Sprintf("{{workflow.status}} == Succeeded")
	case tmv1beta1.ConditionTypeError:
		task.When = fmt.Sprintf("{{workflow.status}} != Succeeded")
	}

	return task
}

func (n *Node) SetOutput() {
	if !n.hasOutput {
		n.TestDefinition.AddSerialStdOutput(false)
		n.hasOutput = true
	}
}

func (n *Node) SetInputSource(node *Node) {
	n.inputSource = node
}

func (n *Node) GetInputSource() *Node {
	return n.inputSource
}

func (n *Node) SetSerial() {
	if !n.isSerial {
		n.TestDefinition.AddSerialStdOutput(true)
		n.isSerial = true
	}
}

func (n *Node) IsSerial() bool {
	return n.isSerial
}

// addTask adds an argo task to the node.AddTask
// Standard artifacts like kubeconfigs and the cloned repository are added as well as a execution condition if specified.
func (n *Node) addTaskOld(lastSerialNode *Node, continueOnError bool) {

	artifacts := []argov1.Artifact{
		{
			Name: "kubeconfigs",
			From: fmt.Sprintf("{{tasks.%s.outputs.artifacts.kubeconfigs}}", lastSerialNode.Template.Name), // todo(schrodit): refactor use task
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
	task := argo.CreateTask(n.TestDefinition.Template.Name, n.TestDefinition.Template.Name, testmachinery.PHASE_RUNNING, continueOnError, n.ParentNames(), artifacts)

	switch n.step.Definition.Condition {
	case tmv1beta1.ConditionTypeSuccess:
		task.When = fmt.Sprintf("{{workflow.status}} == Succeeded")
	case tmv1beta1.ConditionTypeError:
		task.When = fmt.Sprintf("{{workflow.status}} != Succeeded")
	}

}
