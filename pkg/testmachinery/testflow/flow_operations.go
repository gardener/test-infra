package testflow

import (
	"fmt"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow/node"
)

// preprocessTestflow takes a Tesflow and creates a map which maps the unique step name to the step pointer.
func preprocessTestflow(flowID FlowIdentifier, root *node.Node, tf tmv1beta1.TestFlow, loc locations.Locations, globalConfig []*config.Element) (map[string]*Step, map[*testdefinition.TestDefinition]interface{}, map[testdefinition.Location]interface{}, error) {
	stepMap := make(map[string]*Step, 0)
	testdefinitions := make(map[*testdefinition.TestDefinition]interface{}, 0)
	usedLocations := make(map[testdefinition.Location]interface{}, 0)
	for _, step := range tf {
		// todo(schrodit): add validation

		nodes, err := node.CreateNodesFromStep(step, loc, globalConfig, string(flowID))
		if err != nil {
			return nil, nil, nil, err
		}

		for _, n := range nodes {
			testdefinitions[n.TestDefinition] = nil
			usedLocations[n.TestDefinition.Location] = nil
		}
		stepMap[step.Name] = &Step{
			Info:  step,
			Nodes: nodes,
		}
	}
	return stepMap, testdefinitions, usedLocations, nil
}

// CreateInitialDAG creates a DAG by evaluating the dependsOn steps.
func CreateInitialDAG(steps map[string]*Step, root *node.Node) {
	for _, step := range steps {
		if step.Info.DependsOn == nil || len(step.Info.DependsOn) == 0 {
			// add the root node as parent
			step.Nodes.AddParents(root)
			root.AddChildren(step.Nodes...)
			continue
		}

		// go through the list of dependent steps and add them as parent
		for _, dependentStepName := range step.Info.DependsOn {
			dependentStep := steps[dependentStepName]

			step.Nodes.AddParents(dependentStep.Nodes...)
			dependentStep.Nodes.AddChildren(step.Nodes...)
		}
	}
}

// ReorderChildrenOfNodes recursively reorders all children of a nodelist so that serial steps run in serial after parallel nodes.
// Returns nil if successful.
func ReorderChildrenOfNodes(list node.List) node.List {
	children := node.List{}
	for _, item := range list {
		children = append(children, reorderChildrenOfNode(item)...)
	}
	if len(children) == 0 {
		return nil
	}
	return ReorderChildrenOfNodes(children)
}

// reorderSerialNodes reorders all children of a node so that serial steps run in serial after parallel nodes.
// The functions returns the new Children
func reorderChildrenOfNode(root *node.Node) node.List {
	allChildren := root.Children.GetChildren()

	// directly return if there is only one node in the pool
	if len(root.Children) == 0 {
		return allChildren
	}

	serialNodes := node.List{}
	parallelNodes := node.List{}
	for _, item := range root.Children {
		if item.TestDefinition.HasBehavior("serial") {
			serialNodes = append(serialNodes, item)
		} else {
			parallelNodes = append(parallelNodes, item)
		}
	}

	// directly return if there are no serial steps
	if len(serialNodes) == 0 {
		return allChildren
	}

	root.ClearChildren()
	root.AddChildren(parallelNodes...)

	for i, serialNode := range serialNodes {
		if i == 0 {
			parallelNodes.ClearChildren()
			parallelNodes.AddChildren(serialNode)

			serialNode.ClearParents()
			serialNode.AddParents(parallelNodes...)
		} else {
			prevNode := serialNodes[i-1]

			prevNode.ClearChildren()
			prevNode.AddChildren(serialNode)

			serialNode.ClearParents()
			serialNode.AddParents(prevNode)
		}

		if i == len(serialNodes)-1 {
			serialNode.ClearChildren()
			serialNode.AddChildren(allChildren...)

			allChildren.ClearParents()
			allChildren.AddParents(serialNode)
		}
	}

	return allChildren
}

// ApplyOutputNamespaces defines the artifact namesapces for outputs.
// This is done by getting the last serial step and setting is as the current nodes artifact source.
func ApplyOutputNamespaces(steps map[string]*Step) error {
	for _, step := range steps {
		for _, n := range step.Nodes {
			parents := n.Parents
			for len(parents) != 1 {
				parents = parents.GetParents()
				if len(parents) == 0 {
					return fmt.Errorf("no serial parent node can be found for step %s in node %s", n.Name(), step.Info.Name)
				}
			}
			serialNode := parents[0]
			serialNode.SetOutput()
			n.SetInputSource(serialNode)
		}
	}

	return nil
}

// SetSerialNodes evaluates real serial steps and marks them as serial.
// A node is considered serial if all children of the root node point to one child.
func SetSerialNodes(root *node.Node) {
	children := root.Children
	for len(children) != 0 {
		children = children.GetChildren()
		// node is a real serial step if all children of the root node point to one child.
		if len(children) == 1 {
			children[0].SetSerial()
		}
	}
}
