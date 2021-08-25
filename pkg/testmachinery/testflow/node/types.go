package node

import (
	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/util/sets"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

type Set struct {
	set map[*Node]sets.Empty
	// first item of the linked list
	listStart *listItem
	// last item of the linked list
	listEnd *listItem
}

// listItem is a internal structure for a linked list of nodes
type listItem struct {
	node     *Node
	previous *listItem
	next     *listItem
}

// Node is an object that represents a node of the internal DAG representation
type Node struct {
	name     string
	Parents  *Set
	Children *Set

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

	// TODO ??expand struct with mountPath, audience, etc
}
