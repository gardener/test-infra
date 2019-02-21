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
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/argo"
	"github.com/gardener/test-infra/pkg/testmachinery/config"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	apiv1 "k8s.io/api/core/v1"
)

// New takes a testflow definition, test definitions and the global config, and creates a new tesrun representation
func New(flowID FlowIdentifier, tf *tmv1beta1.TestFlow, tl testdefinition.TestDefinitions, globalConfig []*config.Element, prepare *testdefinition.PrepareDefinition) (*Testflow, error) {

	rootNode := NewNode(nil, nil, nil, prepare.TestDefinition, &Step{nil, -1, -1}, flowID)
	rootNode.Task = argo.CreateTask(rootNode.TestDefinition.Template.Name, rootNode.TestDefinition.Template.Name, testmachinery.PHASE_RUNNING, rootNode.GetParentNames(), nil)
	rootNode.TestDefinition.AddConfig(globalConfig)

	flow, err := NewFlow(flowID, rootNode, tf, tl, globalConfig)
	if err != nil {
		return nil, err
	}

	// add used locations to prepare step
	for loc := range flow.usedLocations {
		prepare.AddLocation(loc)
	}
	err = prepare.AddRepositoriesAsArtifacts()
	if err != nil {
		return nil, err
	}

	return &Testflow{tf, flow}, nil
}

// GetTemplates returns all TestDefinitions templates and the DAG of the testrun
func (tf *Testflow) GetTemplates(name string) ([]argov1.Template, error) {
	templates := []argov1.Template{
		{
			Name: name,
			DAG:  tf.Flow.DAG,
		},
	}

	fTemplates, err := tf.Flow.GetTemplates()
	if err != nil {
		return nil, err
	}

	templates = append(templates, fTemplates...)

	return templates, nil
}

// GetLocalVolumes returns all volumes of local TestDefLocations
func (tf *Testflow) GetLocalVolumes() []apiv1.Volume {
	return tf.Flow.GetVolumes()
}
