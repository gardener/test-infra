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
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	apiv1 "k8s.io/api/core/v1"
)

// NewFlow takes a testflow and the global config, and generates the DAG.
// It generates an internal DAG representation and creates the corresponding argo DAG and templates.
func NewFlow(flowID FlowIdentifier, root *Node, tf *tmv1beta1.TestFlow, tl testdefinition.TestDefinitions, globalConfig []*config.Element) (*Flow, error) {

	flow := &Flow{
		ID:              flowID,
		DAG:             &argov1.DAGTemplate{},
		TestFlowRoot:    root,
		globalConfig:    globalConfig,
		testdefinitions: map[*testdefinition.TestDefinition]interface{}{},
		usedLocations:   map[testdefinition.Location]interface{}{},
	}
	flow.addTask(*root.Task)
	flow.testdefinitions[root.TestDefinition] = nil

	lastSerialNode := root
	lastParallelNodes := []*Node{root}
	for row, steps := range *tf {
		currentNodes := []*Node{}
		serialTestdefs := map[*testdefinition.TestDefinition]*Step{}
		var node *Node

		for column, item := range steps {
			s := item
			step := Step{
				Info:   &s,
				Row:    row,
				Column: column,
			}
			testdefinitions, err := tl.GetTestDefinitions(step.Info)
			if err != nil {
				return nil, err
			}

			for _, td := range testdefinitions {
				if td.HasBehavior("serial") {
					serialTestdefs[td] = &step
					continue
				}
				node = flow.addNewNode(lastParallelNodes, lastSerialNode, &step, td)
				currentNodes = append(currentNodes, node)
			}
		}

		if isSerialStep(steps) {
			node.TestDefinition.AddSerialStdOutput()
			lastSerialNode = node
		}

		// when a label is specified and no corresponding testdefs can be found 'current nodes' are empty.
		// Therefore, lastParallelNodes should point to the nodes before.
		if len(currentNodes) > 0 {
			lastParallelNodes = currentNodes
		}

		for serialTestDef, step := range serialTestdefs {
			node = flow.addNewNode(lastParallelNodes, lastSerialNode, step, serialTestDef)
			node.TestDefinition.AddSerialStdOutput()

			currentNodes = []*Node{node}
			lastParallelNodes = currentNodes
			lastSerialNode = node
		}
	}

	return flow, nil
}

// GetTemplates returns all TestDefinitions templates and the DAG of the testrun
func (f *Flow) GetTemplates() ([]argov1.Template, error) {
	var templates []argov1.Template
	for td := range f.testdefinitions {
		templates = append(templates, *td.Template)
	}

	return templates, nil
}

// GetLocalVolumes returns all volumes of local TestDefLocations
func (f *Flow) GetLocalVolumes() []apiv1.Volume {
	var volumes []apiv1.Volume
	for loc := range f.usedLocations {
		if loc.Type() == tmv1beta1.LocationTypeLocal {
			local := loc.(*testdefinition.LocalLocation)
			volumes = append(volumes, local.GetVolume())
		}
	}

	return volumes
}

// Iterate iterates over the flow's levels and returns their nodes.
func (f *Flow) Iterate() <-chan []*Node {
	c := make(chan []*Node)
	go func() {
		currentNode := f.TestFlowRoot
		for len(currentNode.Children) > 0 {
			c <- currentNode.Children
			currentNode = currentNode.Children[0]
		}
		close(c)
	}()
	return c
}

// GetStatus returns the status of all nodes of the current flow.
func (f *Flow) GetStatus() [][]*tmv1beta1.TestflowStepStatus {
	status := [][]*tmv1beta1.TestflowStepStatus{}
	for level := range f.Iterate() {
		stepStatus := []*tmv1beta1.TestflowStepStatus{}
		for _, node := range level {
			stepStatus = append(stepStatus, node.Status)
		}
		status = append(status, stepStatus)
	}
	return status
}

func (f *Flow) addNewNode(lastParallelNodes []*Node, lastSerialNode *Node, step *Step, td *testdefinition.TestDefinition) *Node {
	node := NewNode(lastParallelNodes, lastSerialNode, f.TestFlowRoot, td, step, f.ID)
	td.AddConfig(config.New(step.Info.Config))
	td.AddConfig(f.globalConfig)
	f.addTask(*node.Task)

	f.testdefinitions[td] = nil
	f.usedLocations[td.Location] = nil

	return node
}

func (f *Flow) addTask(task argov1.DAGTask) {
	f.DAG.Tasks = append(f.DAG.Tasks, task)
}

func isSerialStep(steps []tmv1beta1.TestflowStep) bool {
	// TODO: refactor for better check of testStep type
	return (len(steps) == 1 && steps[0].Name != "")
}
