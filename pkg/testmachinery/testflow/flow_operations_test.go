package testflow_test

import (
	"testing"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow/node"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testflow Suite")
}

var _ = Describe("flow operations", func() {
	Context("create initial dag", func() {
		It("should set the root node as parent of all steps with no dependencies", func() {
			rootNode := testNode("root", nil, &defaultTestDef)
			A := testNode("A", nil, &defaultTestDef)
			B := testNode("B", nil, &defaultTestDef)
			steps := map[string]*testflow.Step{
				"A": {
					Info:  testDAGStep([]string{}),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.NewSet(B),
				},
			}
			testflow.CreateInitialDAG(steps, rootNode)

			Expect(A.Parents).To(HaveLen(1))
			Expect(A.Parents).To(HaveKey(rootNode))

			Expect(rootNode.Children).To(HaveLen(1))
			Expect(rootNode.Children).To(HaveKey(A))
		})

		It("should set dependent nodes as parents", func() {
			rootNode := testNode("root", nil, &defaultTestDef)
			A := testNode("A", nil, &defaultTestDef)
			B := testNode("B", nil, &defaultTestDef)
			steps := map[string]*testflow.Step{
				"A": {
					Info:  testDAGStep([]string{}),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.NewSet(B),
				},
			}
			testflow.CreateInitialDAG(steps, rootNode)

			Expect(B.Parents).To(HaveLen(1))
			Expect(B.Parents).To(HaveKey(A))

			Expect(A.Children).To(HaveLen(1))
			Expect(A.Children).To(HaveKey(B))
		})

		It("should set multiple dependent nodes as parents", func() {
			rootNode := testNode("A", nil, &defaultTestDef)
			A := testNode("A", nil, &defaultTestDef)
			B := testNode("B", nil, &defaultTestDef)
			C := testNode("C", nil, &defaultTestDef)
			D := testNode("D", nil, &defaultTestDef)
			steps := map[string]*testflow.Step{
				"A": {
					Info:  testDAGStep([]string{}),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.NewSet(B),
				},
				"C": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.NewSet(C),
				},
				"D": {
					Info:  testDAGStep([]string{"B", "C"}),
					Nodes: node.NewSet(D),
				},
			}
			testflow.CreateInitialDAG(steps, rootNode)

			Expect(A.Children).To(HaveLen(2))
			Expect(A.Children).To(HaveKey(B))
			Expect(A.Children).To(HaveKey(C))

			Expect(B.Parents).To(HaveLen(1))
			Expect(B.Parents).To(HaveKey(A))

			Expect(C.Parents).To(HaveLen(1))
			Expect(C.Parents).To(HaveKey(A))

			Expect(D.Parents).To(HaveLen(2))
			Expect(D.Parents).To(HaveKey(B))
			Expect(D.Parents).To(HaveKey(C))
		})

	})

	Context("serial steps", func() {
		It("should reorder a DAG with 1 parallel and 1 serial step", func() {
			A := testNode("A", nil, &defaultTestDef)
			Bs := testNode("Bs", node.NewSet(A), &serialTestDef)
			C := testNode("C", node.NewSet(A), &defaultTestDef)
			D := testNode("D", node.NewSet(Bs, C), &defaultTestDef)

			Expect(testflow.ReorderChildrenOfNodes(node.NewSet(A))).To(BeNil())

			Expect(A.Children).To(HaveLen(1))
			Expect(A.Children).To(HaveKey(C))

			Expect(C.Children).To(HaveLen(1))
			Expect(C.Children).To(HaveKey(Bs))

			Expect(Bs.Children).To(HaveLen(1))
			Expect(Bs.Children).To(HaveKey(D))

			Expect(D.Parents).To(HaveLen(1))
			Expect(D.Parents).To(HaveKey(Bs))
		})

		It("should reorder a DAG with 2 parallel and 2 serial step", func() {
			A := testNode("A", nil, &defaultTestDef)
			Bs := testNode("Bs", node.NewSet(A), &serialTestDef)
			C := testNode("C", node.NewSet(A), &defaultTestDef)
			D := testNode("D", node.NewSet(A), &defaultTestDef)
			Es := testNode("Es", node.NewSet(A), &serialTestDef)
			F := testNode("F", node.NewSet(Bs, C, D, Es), &defaultTestDef)

			Expect(testflow.ReorderChildrenOfNodes(node.NewSet(A))).To(BeNil())

			Expect(A.Children).To(HaveLen(2))
			Expect(A.Children).To(HaveKey(C))
			Expect(A.Children).To(HaveKey(D))

			Expect(C.Children).To(HaveLen(1))
			Expect(C.Children).To(Or(HaveKey(Bs), HaveKey(Es)))
			Expect(D.Children).To(HaveLen(1))
			Expect(D.Children).To(Or(HaveKey(Bs), HaveKey(Es)))

			Expect(Bs.Children).To(HaveLen(1))
			Expect(Bs.Children).To(Or(HaveKey(Es), HaveKey(F)))
			Expect(Es.Children).To(HaveLen(1))
			Expect(Es.Children).To(Or(HaveKey(Bs), HaveKey(F)))

			Expect(F.Parents).To(HaveLen(1))
			Expect(F.Parents).To(Or(HaveKey(Bs), HaveKey(Es)))
		})

		It("should change a reordered DAG", func() {
			A := testNode("A", nil, &defaultTestDef)
			Bs := testNode("Bs", node.NewSet(A), &serialTestDef)
			C := testNode("C", node.NewSet(A), &defaultTestDef)
			D := testNode("D", node.NewSet(Bs, C), &defaultTestDef)

			Expect(testflow.ReorderChildrenOfNodes(node.NewSet(A))).To(BeNil())
			Expect(testflow.ReorderChildrenOfNodes(node.NewSet(A))).To(BeNil())

			Expect(len(A.Children)).To(Equal(1))
			Expect(A.Children).To(HaveKey(C))

			Expect(len(C.Children)).To(Equal(1))
			Expect(C.Children).To(HaveKey(Bs))

			Expect(len(Bs.Children)).To(Equal(1))
			Expect(Bs.Children).To(HaveKey(D))

			Expect(len(D.Parents)).To(Equal(1))
			Expect(D.Parents).To(HaveKey(Bs))

		})

		It("should not reorder a DAG with 1 serial step", func() {
			A := testNode("A", nil, &defaultTestDef)
			Bs := testNode("Bs", node.NewSet(A), &serialTestDef)

			Expect(testflow.ReorderChildrenOfNodes(node.NewSet(A))).To(BeNil())

			Expect(A.Children).To(HaveLen(1))
			Expect(A.Children).To(HaveKey(Bs))

			Expect(Bs.Children).To(HaveLen(0))

		})
	})

	Context("Apply namespaces", func() {
		It("should set the last serial parent as artifact source", func() {
			rootNode := testNode("root", node.NewSet(), &defaultTestDef)
			A := testNode("A", node.NewSet(rootNode), &defaultTestDef)
			B := testNode("B", node.NewSet(A), &defaultTestDef)
			C := testNode("C", node.NewSet(A), &defaultTestDef)
			D := testNode("D", node.NewSet(B, C), &defaultTestDef)
			steps := map[string]*testflow.Step{
				"A": {
					Info:  testDAGStep([]string{}),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.NewSet(B),
				},
				"C": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.NewSet(C),
				},
				"D": {
					Info:  testDAGStep([]string{"B", "C"}),
					Nodes: node.NewSet(D),
				},
			}

			Expect(testflow.ApplyOutputScope(steps)).ToNot(HaveOccurred())

			Expect(rootNode.HasOutput()).To(BeTrue())
			Expect(rootNode.TestDefinition.Template.Outputs.Artifacts).To(ContainElement(testdefinition.GetStdOutputArtifacts(false)[0]))
			Expect(rootNode.TestDefinition.Template.Outputs.Artifacts).To(ContainElement(testdefinition.GetStdOutputArtifacts(false)[1]))
			Expect(rootNode.TestDefinition.Template.Outputs.Artifacts).To(HaveLen(2))
			Expect(A.GetInputSource()).To(Equal(rootNode))

			Expect(B.GetInputSource()).To(Equal(A))
			Expect(C.GetInputSource()).To(Equal(A))
			Expect(D.GetInputSource()).To(Equal(A))
		})

	})

	Context("serial nodes", func() {
		It("should mark real serial nodes as serial", func() {
			A := testNode("A", node.NewSet(), &defaultTestDef)
			B := testNode("B", node.NewSet(A), &defaultTestDef)
			C := testNode("C", node.NewSet(A), &defaultTestDef)
			D := testNode("D", node.NewSet(B, C), &defaultTestDef)

			testflow.SetSerialNodes(A)

			Expect(A.IsSerial()).To(BeFalse())
			Expect(B.IsSerial()).To(BeFalse())
			Expect(C.IsSerial()).To(BeFalse())
			Expect(D.IsSerial()).To(BeTrue())
		})

		It("should mark all nodes as serial", func() {
			A := testNode("A", node.NewSet(), &defaultTestDef)
			B := testNode("B", node.NewSet(A), &defaultTestDef)
			C := testNode("C", node.NewSet(B), &defaultTestDef)
			D := testNode("D", node.NewSet(C), &defaultTestDef)

			testflow.SetSerialNodes(A)

			Expect(A.IsSerial()).To(BeFalse())
			Expect(B.IsSerial()).To(BeTrue())
			Expect(C.IsSerial()).To(BeTrue())
			Expect(D.IsSerial()).To(BeTrue())
		})

		It("should mark real serial nodes as serial in a huge DAG", func() {
			A := testNode("A", node.NewSet(), &defaultTestDef)
			B := testNode("B", node.NewSet(A), &defaultTestDef)
			C := testNode("C", node.NewSet(A), &defaultTestDef)
			D := testNode("D", node.NewSet(B), &defaultTestDef)
			E := testNode("E", node.NewSet(C), &defaultTestDef)
			F := testNode("F", node.NewSet(C), &defaultTestDef)
			G := testNode("D", node.NewSet(D, E, F), &defaultTestDef)
			H := testNode("H", node.NewSet(G), &defaultTestDef)
			I := testNode("I", node.NewSet(G), &defaultTestDef)
			J := testNode("J", node.NewSet(H, I), &defaultTestDef)
			K := testNode("K", node.NewSet(J), &defaultTestDef)

			testflow.SetSerialNodes(A)

			Expect(A.IsSerial()).To(BeFalse())
			Expect(B.IsSerial()).To(BeFalse())
			Expect(C.IsSerial()).To(BeFalse())
			Expect(D.IsSerial()).To(BeFalse())
			Expect(E.IsSerial()).To(BeFalse())
			Expect(F.IsSerial()).To(BeFalse())
			Expect(G.IsSerial()).To(BeTrue())
			Expect(H.IsSerial()).To(BeFalse())
			Expect(I.IsSerial()).To(BeFalse())
			Expect(J.IsSerial()).To(BeTrue())
			Expect(K.IsSerial()).To(BeTrue())
		})
	})
})

func testNode(name string, parents node.Set, td *testdefinition.TestDefinition) *node.Node {
	if parents == nil {
		parents = node.NewSet()
	}
	n := &node.Node{
		Parents:        parents,
		Children:       node.NewSet(),
		TestDefinition: td,
	}

	for parent := range parents {
		parent.AddChildren(n)
	}

	td.SetName(name)

	return n
}

func testDAGStep(dependencies []string) *v1beta1.DAGStep {
	return &v1beta1.DAGStep{
		DependsOn: dependencies,
	}
}

var serialTestDef = testdefinition.TestDefinition{
	Info: &v1beta1.TestDefinition{
		Spec: v1beta1.TestDefSpec{
			Behavior: []string{"serial"},
		},
	},
	Template: &argov1.Template{},
}
var defaultTestDef = testdefinition.TestDefinition{
	Info:     &v1beta1.TestDefinition{},
	Template: &argov1.Template{},
}
