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

		for n := range nodes {
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
			root.AddChildren(step.Nodes.List()...)
			continue
		}

		// go through the list of dependent steps and add them as parent
		addDependentStepsAsParent(steps, step)
	}
}

func addDependentStepsAsParent(steps map[string]*Step, step *Step) {
	for _, dependentStepName := range step.Info.DependsOn {
		dependentStep := steps[dependentStepName]

		step.Nodes.AddParents(dependentStep.Nodes.List()...)
		dependentStep.Nodes.AddChildren(step.Nodes.List()...)
	}
}

// ReorderChildrenOfNodes recursively reorders all children of a nodelist so that serial steps run in serial after parallel nodes.
// Returns nil if successful.
func ReorderChildrenOfNodes(list node.Set) node.Set {
	children := make(node.Set, 0)
	for item := range list {
		// use k8s sets
		children.Add(reorderChildrenOfNode(item).List()...)
	}
	if len(children) == 0 {
		return nil
	}
	return ReorderChildrenOfNodes(children)
}

// reorderSerialNodes reorders all children of a node so that serial steps run in serial after parallel nodes.
// The functions returns the new Children
func reorderChildrenOfNode(root *node.Node) node.Set {
	grandChildren := root.Children.GetChildren()

	// directly return if there is only one node in the pool
	if len(root.Children) == 1 {
		// todo: write test for special case
		return root.Children
	}

	serialNodes := make(node.Set, 0)
	parallelNodes := make(node.Set, 0)
	for item := range root.Children {
		if item.TestDefinition.HasBehavior("serial") {
			serialNodes.Add(item)
		} else {
			parallelNodes.Add(item)
		}
	}

	// directly return if there are no serial steps
	if len(serialNodes) == 0 {
		return root.Children
	}

	root.ClearChildren()
	root.AddChildren(parallelNodes.List()...)

	for i, serialNode := range serialNodes.List() {
		if i == 0 {
			parallelNodes.ClearChildren()
			parallelNodes.AddChildren(serialNode)

			serialNode.ClearParents()
			serialNode.AddParents(parallelNodes.List()...)
		} else {
			prevNode := serialNodes.List()[i-1]

			prevNode.ClearChildren()
			prevNode.AddChildren(serialNode)

			serialNode.ClearParents()
			serialNode.AddParents(prevNode)
		}

		if i == len(serialNodes)-1 {
			serialNode.ClearChildren()
			serialNode.AddChildren(grandChildren.List()...)

			grandChildren.ClearParents()
			grandChildren.AddParents(serialNode)
		}
	}

	return grandChildren
}

// ApplyOutputScope defines the artifact scopes for outputs.
// This is done by getting the last serial step and setting is as the current nodes artifact source.
func ApplyOutputScope(steps map[string]*Step) error {
	for _, step := range steps {
		for n := range step.Nodes {
			parents := n.Parents
			for len(parents) != 1 {
				parents = parents.GetParents()
				if len(parents) == 0 {
					return fmt.Errorf("no serial parent node can be found for step %s in node %s", n.Name(), step.Info.Name)
				}
			}
			// rename
			outputSourceNode := parents.List()[0]
			outputSourceNode.EnableOutput()
			n.SetInputSource(outputSourceNode)
		}
	}

	return nil
}

// SetSerialNodes evaluates real serial steps and marks them as serial.
// A node is considered serial if all children of the root node point to one child.
func SetSerialNodes(root *node.Node) {
	children := root.Children
	for len(children) != 0 {
		// node is a real serial step if all children of the root node point to one child.
		if len(children) == 1 {
			children.List()[0].SetSerial()
		}
		children = children.GetChildren()
	}
}
