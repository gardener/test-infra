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
)

// FlowIdentifier is the flow identifier
type FlowIdentifier string

const (
	FlowIDTest FlowIdentifier = "testflow"
	FlowIDExit FlowIdentifier = "exit"
)

// Testflow is an object containing informations about the testflow of a testrun
type Testflow struct {
	Info *tmv1beta1.TestFlow
	Flow *Flow
}

// Flow represents the internal DAG.
type Flow struct {
	ID           FlowIdentifier
	DAG          *argov1.DAGTemplate
	TestFlowRoot *Node

	testdefinitions map[*testdefinition.TestDefinition]interface{}
	usedLocations   map[testdefinition.Location]interface{}
	globalConfig    []*config.Element
}

// Step is a TestflowStep with its specific Row and Column in the testflow.
type Step struct {
	Info   *tmv1beta1.TestflowStep
	Row    int
	Column int
}

// Node is an object that represents a node of the internal DAG representation
type Node struct {
	Parents  []*Node
	Children []*Node
	Position FlowPosition

	Task           *argov1.DAGTask
	TestDefinition *testdefinition.TestDefinition
	Template       *argov1.Template

	step *tmv1beta1.TestflowStep
}

// FlowPosition is the position in the execution flow
type FlowPosition struct {
	Type   FlowIdentifier
	Row    int64
	Column int64
}
