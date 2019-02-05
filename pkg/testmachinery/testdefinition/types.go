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

package testdefinition

import (
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// TestDefinition represents a TestDefinition which was fetched from locations.
type TestDefinition struct {
	Info     *tmv1beta1.TestDefinition
	TaskName string
	Location Location
	FileName string
	Template *argov1.Template
}

// PrepareDefinition is the TestDefinition of the prepare step to initiliaze the setup.
type PrepareDefinition struct {
	TestDefinition *TestDefinition
	repositories   []*PrepareRepository
}

// Location is an interface for different testDefLocation types like git or local
type Location interface {
	// SetTestDefs adds Testdefinitions to the map.
	SetTestDefs(map[string]*TestDefinition) error
	// Type returns the tmv1beta1.LocationType type.
	Type() tmv1beta1.LocationType
	// Name returns the unique name of the location.
	Name() string
	// GetLocation returns the original TestLocation object
	GetLocation() *tmv1beta1.TestLocation
}

// TestDefinitions is a interface which provides function for receiving TestDefinitions that are fetched from testDefLocations.
type TestDefinitions interface {
	GetTestDefinitions(*tmv1beta1.TestflowStep) ([]*TestDefinition, error)
}
type testDefinitions struct {
	Info            []tmv1beta1.TestLocation
	TestDefinitions map[string]*TestDefinition
}

// PrepareRepository is passed as a json array to the prepare step.
type PrepareRepository struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Revision string `json:"revision"`
}
