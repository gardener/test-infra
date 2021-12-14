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
	argov1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/testmachinery/locations/location"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow/node"
)

// NewFlow takes a testflow and the global config, and generates the DAG.
// It generates an internal DAG representation and creates the corresponding argo DAG and templates.
func NewFlow(flowID FlowIdentifier, root *node.Node, tf tmv1beta1.TestFlow, loc locations.Locations, globalConfig []*config.Element) (*Flow, error) {

	steps, testdefinitions, usedLocations, err := preprocessTestflow(flowID, root, tf, loc, globalConfig)
	if err != nil {
		return nil, err
	}

	flow := &Flow{
		ID:              flowID,
		Root:            root,
		steps:           steps,
		globalConfig:    globalConfig,
		testdefinitions: testdefinitions,
		usedLocations:   usedLocations,
	}
	flow.testdefinitions[root.TestDefinition] = nil

	// Go through all steps and create the initial DAG
	CreateInitialDAG(steps, root)

	// Reorder the dag so that tests with serial behavior run in serial within their sub DAG.
	ReorderChildrenOfNodes(node.NewSet(root))

	// Determine kubeconfigs namespaces
	// which means to determine what kubeconfig artifact should be mounted to a specific node
	if err := ApplyOutputScope(steps); err != nil {
		return nil, err
	}

	ApplyConfigScope(steps)

	// Determine real serial steps
	SetSerialNodes(root)

	return flow, nil
}

// GetTemplates returns all TestDefinitions templates and the DAG of the testrun
func (f *Flow) GetTemplates() ([]argov1.Template, error) {
	var templates []argov1.Template
	for td := range f.testdefinitions {
		newTemplate, err := td.GetTemplate()
		if err != nil {
			return nil, err
		}
		templates = append(templates, *newTemplate)
	}

	return templates, nil
}

// GetVolumes returns all volumes of local TestDefLocations and
// used configuration with a reference to a secret or configuration.
func (f *Flow) GetVolumes() []corev1.Volume {
	var volumes []corev1.Volume
	for loc := range f.usedLocations {
		if loc.Type() == tmv1beta1.LocationTypeLocal {
			local := loc.(*location.LocalLocation)
			volumes = append(volumes, local.GetVolume())
		}
	}

	volumeSet := make(map[string]bool)

	for _, vol := range f.Root.TestDefinition.Volumes {
		if _, ok := volumeSet[vol.Name]; !ok {
			volumes = append(volumes, vol)
			volumeSet[vol.Name] = true
		}
	}
	for n := range f.Iterate() {
		for _, vol := range n.TestDefinition.Volumes {
			if _, ok := volumeSet[vol.Name]; !ok {
				volumes = append(volumes, vol)
				volumeSet[vol.Name] = true
			}
		}
	}

	return volumes
}

// Iterate iterates over the flow and returns their nodes.
func (f *Flow) Iterate() <-chan *node.Node {
	c := make(chan *node.Node)
	go func() {
		c <- f.Root
		for _, step := range f.steps {
			for n := range step.Nodes.Iterate() {
				c <- n
			}
		}
		close(c)
	}()
	return c
}

func (f *Flow) GetDAGTemplate(phase testmachinery.Phase, trustedTokenMounts, untrustedTokenMounts []node.ProjectedTokenMount) *argov1.DAGTemplate {
	dag := &argov1.DAGTemplate{}

	for n := range f.Iterate() {
		dag.Tasks = append(dag.Tasks, n.Task(phase, trustedTokenMounts, untrustedTokenMounts)...)
	}

	return dag
}

// GetStatuses returns the status of all nodes of the current flow.
func (f *Flow) GetStatuses() []*tmv1beta1.StepStatus {
	status := make([]*tmv1beta1.StepStatus, 0)
	for n := range f.Iterate() {
		status = append(status, n.Status())
	}
	// remove root element from status as this is the tm prepare step
	return status[1:]
}
