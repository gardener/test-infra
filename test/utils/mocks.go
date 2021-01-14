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

package utils

import (
	"fmt"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

type LocationsMock struct {
	TestDefinitions        []*testdefinition.TestDefinition
	StepToTestDefinitions  map[string][]*testdefinition.TestDefinition
	GetTestDefinitionsFunc func(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error)
}

var _ locations.Locations = &LocationsMock{}

func (t *LocationsMock) GetTestDefinitions(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error) {
	if t.GetTestDefinitionsFunc != nil {
		return t.GetTestDefinitionsFunc(step)
	}
	if t.StepToTestDefinitions != nil {
		td, ok := t.StepToTestDefinitions[step.Name]
		if !ok {
			return nil, fmt.Errorf("no testdefinitions found for step %q", step.Name)
		}
		return td, nil
	}
	if t.TestDefinitions != nil {
		return t.TestDefinitions, nil
	}
	return nil, nil
}

var EmptyMockLocation = &LocationsMock{
	GetTestDefinitionsFunc: func(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error) {
		return []*testdefinition.TestDefinition{}, nil
	},
}

type TDLocationMock struct{}

var _ testdefinition.Location = &TDLocationMock{}

func (l *TDLocationMock) Name() string {
	return "locationmock"
}

func (l *TDLocationMock) Type() tmv1beta1.LocationType {
	return "mock"
}

func (l *TDLocationMock) GitInfo() testdefinition.GitInfo {
	return testdefinition.GitInfo{}
}

func (l *TDLocationMock) SetTestDefs(_ map[string]*testdefinition.TestDefinition) error {
	return nil
}

func (l *TDLocationMock) GetLocation() *tmv1beta1.TestLocation {
	return nil
}

/**
TestDefinition helper
*/

// TestDef creates a new empty testdefinition with a step name
func TestDef(name string) *testdefinition.TestDefinition {
	td := testdefinition.NewEmpty()
	td.Info.Name = name
	return td
}

// SerialTestDef creates a new serial testdefinition with a name
func SerialTestDef(name string) *testdefinition.TestDefinition {
	td := TestDef(name)
	td.Info.Spec.Behavior = []string{tmv1beta1.SerialBehavior}
	return td
}

// DisruptiveTestDef creates a new disruptive testdefinition with a name
func DisruptiveTestDef(name string) *testdefinition.TestDefinition {
	td := TestDef(name)
	td.Info.Spec.Behavior = []string{tmv1beta1.DisruptiveBehavior}
	return td
}

// TestDefWithConfig adds a config to the given testdefinition and returns it
func TestDefWithConfig(td *testdefinition.TestDefinition, cfgs []tmv1beta1.ConfigElement) *testdefinition.TestDefinition {
	td.AddConfig(config.New(cfgs, config.LevelTestDefinition))
	return td
}
