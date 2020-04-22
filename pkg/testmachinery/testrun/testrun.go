// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package testrun

import (
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/argo"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/testmachinery/prepare"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	"github.com/go-logr/logr"
)

// New takes a testrun crd and creates a new Testrun representation.
// It fetches testruns from specified testdeflocations and generates a testflow object.
func New(log logr.Logger, tr *tmv1beta1.Testrun) (*Testrun, error) {
	locs, err := locations.NewLocations(log, tr.Spec)
	if err != nil {
		return nil, err
	}

	globalConfig := config.New(tr.Spec.Config, config.LevelGlobal)
	globalConfig = append(globalConfig, config.NewElement(createTestrunIDConfig(tr.Name), config.LevelGlobal))

	kubeconfigs, secrets, err := ParseKubeconfigs(tr)
	if err != nil {
		return nil, err
	}

	// create initial prepare step
	prepareDef, err := prepare.New("Prepare", false, true)
	if err != nil {
		return nil, err
	}
	prepareDef.TestDefinition.AddConfig(kubeconfigs)
	tf, err := testflow.New(testflow.FlowIDTest, tr.Spec.TestFlow, locs, globalConfig, prepareDef)
	if err != nil {
		return nil, err
	}

	postPrepareDef, err := prepare.New("PostPrepare", true, false)
	if err != nil {
		return nil, err
	}
	postPrepareDef.TestDefinition.AddConfig(kubeconfigs)
	onExitFlow, err := testflow.New(testflow.FlowIDExit, tr.Spec.OnExit, locs, globalConfig, postPrepareDef)
	if err != nil {
		return nil, err
	}

	return &Testrun{
		Info:            tr,
		Testflow:        tf,
		OnExitTestflow:  onExitFlow,
		HelperResources: secrets,
	}, nil
}

// GetWorkflow returns the argo workflow object of this testrun.
func (tr *Testrun) GetWorkflow(name, namespace string, pullImageSecretNames []string) (*argov1.Workflow, error) {
	testrunName := "testrun"
	onExitName := "exit-handler"

	templates, err := tr.Testflow.GetTemplates(testrunName, testmachinery.PhaseRunning)
	if err != nil {
		return nil, err
	}
	onExitTemplates, err := tr.OnExitTestflow.GetTemplates(onExitName, testmachinery.PhaseExit)
	if err != nil {
		return nil, err
	}

	volumes := tr.Testflow.GetLocalVolumes()
	onExitVolumes := tr.OnExitTestflow.GetLocalVolumes()

	return argo.CreateWorkflow(name, namespace, testrunName, onExitName, append(templates, onExitTemplates...), append(volumes, onExitVolumes...), tr.Info.Spec.TTLSecondsAfterFinished, pullImageSecretNames)
}

func createTestrunIDConfig(id string) *tmv1beta1.ConfigElement {
	return &tmv1beta1.ConfigElement{
		Type:  tmv1beta1.ConfigTypeEnv,
		Name:  testmachinery.TM_TESTRUN_ID_NAME,
		Value: id,
	}
}
