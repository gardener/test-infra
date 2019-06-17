package node

import (
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

type Set map[*Node]empty
type empty struct{}

// Node is an object that represents a node of the internal DAG representation
type Node struct {
	name     string
	Parents  Set
	Children Set

	hasOutput   bool
	inputSource *Node

	// Indicates if the node is real serial
	// This would result in global outputs
	isSerial bool

	TestDefinition *testdefinition.TestDefinition
	Template       *argov1.Template

	// metadata
	step *tmv1beta1.DAGStep
	flow string
}
