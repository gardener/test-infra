// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testflow

import (
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow/node"
)

// FlowIdentifier is the flow identifier
type FlowIdentifier string

const (
	// FlowIDTest represents the flow identifier of the main testflow "spec.testflow"
	FlowIDTest FlowIdentifier = "testflow"
	// FlowIDExit represents the flow identifier of the onExit testflow "spec.onExit"
	FlowIDExit FlowIdentifier = "exit"
)

// Testflow is an object containing informations about the testflow of a testrun
type Testflow struct {
	Info tmv1beta1.TestFlow
	Flow *Flow
}

// Flow represents the internal DAG.
type Flow struct {
	ID   FlowIdentifier
	Root *node.Node

	steps           map[string]*Step
	testdefinitions map[*testdefinition.TestDefinition]interface{}
	usedLocations   map[testdefinition.Location]interface{}
	globalConfig    []*config.Element
}

// Step is a StepDefinition with its specific Row and Column in the testflow.
type Step struct {
	Info  *tmv1beta1.DAGStep
	Nodes *node.Set
}
