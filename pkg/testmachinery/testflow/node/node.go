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

	argov1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/argo"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

// CreateNodesFromStep creates new nodes from a step and adds default configuration
func CreateNodesFromStep(step *tmv1beta1.DAGStep, loc locations.Locations, globalConfig []*config.Element, flowID string) (*Set, error) {
	testdefinitions, err := loc.GetTestDefinitions(step.Definition)
	if err != nil {
		return nil, err
	}

	nodes := NewSet()
	for _, td := range testdefinitions {
		node := NewNode(td, step, flowID)
		td.AddConfig(config.New(step.Definition.Config, config.LevelStep))
		td.AddConfig(globalConfig)
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
		name:           name,
		TestDefinition: td,
		step:           step.DeepCopy(),
		flow:           flow,
		Parents:        NewSet(),
		Children:       NewSet(),
	}

	if td.HasBehavior(tmv1beta1.DisruptiveBehavior) {
		node.step.Definition.ContinueOnError = false
	}

	return node
}

// NewEmpty creates and new empty node with a name.
func NewEmpty(name string) *Node {
	return &Node{
		name:     name,
		Parents:  NewSet(),
		Children: NewSet(),
	}
}

// AddChildren adds Nodes as children
func (n *Node) AddChildren(children ...*Node) *Node {
	n.Children.Add(children...)
	return n
}

// RemoveChild removes a node from the current node's children
func (n *Node) RemoveChild(child *Node) *Node {
	child.RemoveParent(n)
	n.Children.Remove(child)
	return n
}

// RemoveChildren removes all nodes from the current node's children
func (n *Node) RemoveChildren(children ...*Node) *Node {
	for _, child := range children {
		n.RemoveChild(child)
	}
	return n
}

// ClearChildren removes all children from the current node
func (n *Node) ClearChildren() *Node {
	for child := range n.Children.Iterate() {
		child.RemoveParent(n)
	}
	n.Children = NewSet()
	return n
}

// AddParents adds nodes as parents.
func (n *Node) AddParents(parents ...*Node) {
	n.Parents.Add(parents...)
}

// ClearParent removes a node from the current node's parents
func (n *Node) RemoveParent(parent *Node) *Node {
	n.Parents.Remove(parent)
	return n
}

// ClearParents removes all parents from the current node
func (n *Node) ClearParents() *Node {
	n.Parents = NewSet()
	return n
}

// ParentNames returns the names of all parent nodes
func (n *Node) ParentNames() []string {
	names := make([]string, 0)
	for parent := range n.Parents.Iterate() {
		names = append(names, parent.Name())
	}
	return names
}

// Name returns the unique name of the node's task
func (n *Node) Name() string {
	return n.TestDefinition.GetName()
}

// Step returns the origin step of the node
func (n *Node) Step() *tmv1beta1.DAGStep {
	return n.step
}

// SetStep set the step of a node
func (n *Node) SetStep(step *tmv1beta1.DAGStep) {
	n.step = step
}

// Task returns the argo task definition for the node.
func (n *Node) Task(phase testmachinery.Phase) []argov1.DAGTask {
	tasks := make([]argov1.DAGTask, 0)
	var task argov1.DAGTask
	if n.step.Pause != nil && n.step.Pause.Enabled {
		suspendTask := argo.CreateSuspendTask(n.TestDefinition.GetName(), n.ParentNames())
		task = argo.CreateTask(n.TestDefinition.GetName(), n.TestDefinition.GetName(), string(phase), n.step.Definition.ContinueOnError, []string{suspendTask.Name}, n.GetOrDetermineArtifacts())
		tasks = append(tasks, suspendTask)
	} else {
		task = argo.CreateTask(n.TestDefinition.GetName(), n.TestDefinition.GetName(), string(phase), n.step.Definition.ContinueOnError, n.ParentNames(), n.GetOrDetermineArtifacts())
	}

	switch n.step.Definition.Condition {
	case tmv1beta1.ConditionTypeSuccess:
		task.When = "{{workflow.status}} == Succeeded"
	case tmv1beta1.ConditionTypeError:
		task.When = "{{workflow.status}} != Succeeded"
	}

	return append(tasks, task)
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
		Annotations: n.step.Annotations,
		Phase:       tmv1beta1.PhaseStatusInit,
		TestDefinition: tmv1beta1.StepStatusTestDefinition{
			Name:                  td.Info.Name,
			Owner:                 td.Info.Spec.Owner,
			Config:                td.GetConfig().RawList(),
			Labels:                td.Info.Spec.Labels,
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
	return n.inputSource == nil
}

func (n *Node) GetOrDetermineArtifacts() []argov1.Artifact {
	artifactsMap := make(map[string]argov1.Artifact)
	if !n.isRootNode() {

		if n.Step().Definition.Untrusted {
			artifactsMap[testmachinery.ArtifactUntrustedKubeconfigs] = argov1.Artifact{
				Name: testmachinery.ArtifactUntrustedKubeconfigs,
				From: fmt.Sprintf("{{tasks.%s.outputs.artifacts.%s}}", n.inputSource.Name(), testmachinery.ArtifactUntrustedKubeconfigs),
			}
		} else {
			artifactsMap[testmachinery.ArtifactKubeconfigs] = argov1.Artifact{
				Name: testmachinery.ArtifactKubeconfigs,
				From: fmt.Sprintf("{{tasks.%s.outputs.artifacts.%s}}", n.inputSource.Name(), testmachinery.ArtifactKubeconfigs),
			}
			artifactsMap[testmachinery.ArtifactSharedFolder] = argov1.Artifact{
				Name: testmachinery.ArtifactSharedFolder,
				From: fmt.Sprintf("{{tasks.%s.outputs.artifacts.%s}}", n.inputSource.Name(), testmachinery.ArtifactSharedFolder),
			}
		}

		if n.TestDefinition.Location != nil && n.TestDefinition.Location.Type() != tmv1beta1.LocationTypeLocal {
			artifactsMap["repo"] = argov1.Artifact{
				Name: "repo",
				From: fmt.Sprintf("{{workflow.outputs.artifacts.%s}}", n.TestDefinition.Location.Name()),
			}
		}
	}

	if n.step.UseGlobalArtifacts {
		if n.Step().Definition.Untrusted {
			artifactsMap[testmachinery.ArtifactUntrustedKubeconfigs] = argov1.Artifact{
				Name: testmachinery.ArtifactUntrustedKubeconfigs,
				From: fmt.Sprintf("{{workflow.outputs.artifacts.%s}}", testmachinery.ArtifactUntrustedKubeconfigs),
			}
		} else {
			artifactsMap[testmachinery.ArtifactKubeconfigs] = argov1.Artifact{
				Name: testmachinery.ArtifactKubeconfigs,
				From: fmt.Sprintf("{{workflow.outputs.artifacts.%s}}", testmachinery.ArtifactKubeconfigs),
			}
			artifactsMap[testmachinery.ArtifactSharedFolder] = argov1.Artifact{
				Name: testmachinery.ArtifactSharedFolder,
				From: fmt.Sprintf("{{workflow.outputs.artifacts.%s}}", testmachinery.ArtifactSharedFolder),
			}
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

// HasOutput indicates if the node has output
func (n *Node) HasOutput() bool {
	return n.hasOutput
}

// SetInputSource sets the input source node for artifacts that are mounted to the test.
func (n *Node) SetInputSource(node *Node) {
	n.inputSource = node
	if !n.step.Definition.Untrusted {
		n.TestDefinition.AddInputArtifacts(testdefinition.GetStdInputArtifacts()...)
	} else {
		n.TestDefinition.AddInputArtifacts(testdefinition.GetUntrustedInputArtifacts()...)
	}
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
