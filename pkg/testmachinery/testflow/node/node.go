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
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/argo"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

// CreateNodesFromStep creates new nodes from a step and adds default configuration
func CreateNodesFromStep(step *tmv1beta1.DAGStep, loc locations.Locations, globalConfig []*config.Element, flowID string) (Set, error) {
	testdefinitions, err := loc.GetTestDefinitions(step.Definition)
	if err != nil {
		return nil, err
	}

	nodes := make(Set, 0)
	for _, td := range testdefinitions {
		node := NewNode(td, step, flowID)
		if err := td.AddConfig(config.New(step.Definition.Config)); err != nil {
			return nil, err
		}
		if err := td.AddConfig(globalConfig); err != nil {
			return nil, err
		}
		nodes.Add(node)
	}
	return nodes, nil
}

// NewNode creates a new TestflowNode for the internal DAG
func NewNode(td *testdefinition.TestDefinition, step *tmv1beta1.DAGStep, flow string) *Node {
	// create hash or unique name for testdefinition + step + flow
	name := GetUniqueName(td, step, flow)
	td.SetName(name)
	node := &Node{
		TestDefinition: td,
		step:           step,
		flow:           flow,
		Parents:        NewSet(),
		Children:       NewSet(),
	}

	return node
}

// AddChildren adds Nodes as children
func (n *Node) AddChildren(children ...*Node) {
	n.Children.Add(children...)
}

// ClearParent removes a node from the current node's children
func (n *Node) RemoveChild(child *Node) {
	n.Children.Remove(child)
}

// ClearChildren removes all children from the current node
func (n *Node) ClearChildren() {
	n.Children = make(Set, 0)
}

// AddParents adds nodes as parents.
func (n *Node) AddParents(parents ...*Node) {
	n.Parents.Add(parents...)
}

// ClearParent removes a node from the current node's parents
func (n *Node) RemoveParent(parent *Node) {
	n.Parents.Remove(parent)
}

// ClearParents removes all parents from the current node
func (n *Node) ClearParents() {
	n.Parents = make(Set, 0)
}

// ParentNames returns the names of all parent nodes
func (n *Node) ParentNames() []string {
	names := make([]string, 0)
	for parent := range n.Parents {
		names = append(names, parent.Name())
	}
	return names
}

// Name return the unique name of the node's task
func (n *Node) Name() string {
	return n.TestDefinition.GetName()
}

// Task returns the argo task definition for the node.
func (n *Node) Task() argov1.DAGTask {
	task := argo.CreateTask(n.TestDefinition.GetName(), n.TestDefinition.GetName(), testmachinery.PHASE_RUNNING, n.step.Definition.ContinueOnError, n.ParentNames(), n.DetermineArtifacts())

	switch n.step.Definition.Condition {
	case tmv1beta1.ConditionTypeSuccess:
		task.When = fmt.Sprintf("{{workflow.status}} == Succeeded")
	case tmv1beta1.ConditionTypeError:
		task.When = fmt.Sprintf("{{workflow.status}} != Succeeded")
	}

	return task
}

// Status returns the status for the test step based in the node.
func (n *Node) Status() *tmv1beta1.StepStatus {
	td := n.TestDefinition
	status := &tmv1beta1.StepStatus{
		Name: n.Name(),
		Position: tmv1beta1.StepStatusPosition{
			DependsOn: n.ParentNames(),
			Flow:      n.flow,
		},
		Phase: tmv1beta1.PhaseStatusInit,
		TestDefinition: tmv1beta1.StepStatusTestDefinition{
			Name:                  td.Info.Metadata.Name,
			Owner:                 td.Info.Spec.Owner,
			RecipientsOnFailure:   td.Info.Spec.RecipientsOnFailure,
			ActiveDeadlineSeconds: td.Info.Spec.ActiveDeadlineSeconds,
		},
	}
	if n.step != nil {
		status.Position.Step = n.step.Name
	}
	if td.Location != nil {
		status.TestDefinition.Location = *td.Location.GetLocation()
	}

	return status
}

func (n *Node) isRootNode() bool {
	return n.inputSource != nil
}

func (n *Node) DetermineArtifacts() []argov1.Artifact {
	artifactsMap := make(map[string]argov1.Artifact)
	if n.isRootNode() {
		artifactsMap["kubeconfigs"] = argov1.Artifact{
			Name: "kubeconfigs",
			From: fmt.Sprintf("{{tasks.%s.outputs.artifacts.kubeconfigs}}", n.inputSource.Name()),
		}
		artifactsMap["sharedFolder"] = argov1.Artifact{
			Name: "sharedFolder",
			From: fmt.Sprintf("{{tasks.%s.outputs.artifacts.sharedFolder}}", n.inputSource.Name()),
		}

		if n.TestDefinition.Location != nil && n.TestDefinition.Location.Type() != tmv1beta1.LocationTypeLocal {
			artifactsMap["repo"] = argov1.Artifact{
				Name: "repo",
				From: fmt.Sprintf("{{workflow.outputs.artifacts.%s}}", n.TestDefinition.Location.Name()),
			}
		}
	}

	if n.step.UseGlobalArtifacts {
		artifactsMap["kubeconfigs"] = argov1.Artifact{
			Name: "kubeconfigs",
			From: "{{workflow.outputs.artifacts.kubeconfigs}}",
		}
		artifactsMap["sharedFolder"] = argov1.Artifact{
			Name: "sharedFolder",
			From: "{{workflow.outputs.artifacts.sharedFolder}}",
		}
	}
	return artifactsMapToList(artifactsMap)
}

func artifactsMapToList(m map[string]argov1.Artifact) []argov1.Artifact {
	list := make([]argov1.Artifact, 0, len(m))
	for _, value := range m {
		list = append(list, value)
	}
	return list
}

// EnableOutput adds std output to the test and marks the node as node with output.
func (n *Node) EnableOutput() {
	if !n.hasOutput {
		n.TestDefinition.AddStdOutput(false)
		n.hasOutput = true
	}
}

// SetInputSource sets the input source node for artifacts that are mounted to the test.
func (n *Node) SetInputSource(node *Node) {
	n.inputSource = node
}

// GetInputSource returns the input source node.
func (n *Node) GetInputSource() *Node {
	return n.inputSource
}

// SetSerial adds global std output to the test and marks the node as serial.
func (n *Node) SetSerial() {
	if !n.isSerial {
		n.TestDefinition.AddStdOutput(true)
		n.isSerial = true
	}
}

// IsSerial returns true if the node is serial.
func (n *Node) IsSerial() bool {
	return n.isSerial
}
