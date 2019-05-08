package testflow_test

import (
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow/node"
	"testing"

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
			rootNode := testNode(nil, &defaultTestDef)
			A := testNode(nil, &defaultTestDef)
			B := testNode(nil, &defaultTestDef)
			steps := map[string]*testflow.Step{
				"A": {
					Info:  testDAGStep([]string{}),
					Nodes: node.List{A},
				},
				"B": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.List{B},
				},
			}
			testflow.CreateInitialDAG(steps, rootNode)

			Expect(A.Parents).To(HaveLen(1))
			Expect(A.Parents).To(ContainElement(rootNode))

			Expect(rootNode.Children).To(HaveLen(1))
			Expect(rootNode.Children).To(ContainElement(A))
		})

		It("should set dependent nodes as parents", func() {
			rootNode := testNode(nil, &defaultTestDef)
			A := testNode(nil, &defaultTestDef)
			B := testNode(nil, &defaultTestDef)
			steps := map[string]*testflow.Step{
				"A": {
					Info:  testDAGStep([]string{}),
					Nodes: node.List{A},
				},
				"B": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.List{B},
				},
			}
			testflow.CreateInitialDAG(steps, rootNode)

			Expect(B.Parents).To(HaveLen(1))
			Expect(B.Parents).To(ContainElement(A))

			Expect(A.Children).To(HaveLen(1))
			Expect(A.Children).To(ContainElement(B))
		})

		It("should set multiple dependent nodes as parents", func() {
			rootNode := testNode(nil, &defaultTestDef)
			A := testNode(nil, &defaultTestDef)
			B := testNode(nil, &defaultTestDef)
			C := testNode(nil, &defaultTestDef)
			D := testNode(nil, &defaultTestDef)
			steps := map[string]*testflow.Step{
				"A": {
					Info:  testDAGStep([]string{}),
					Nodes: node.List{A},
				},
				"B": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.List{B},
				},
				"C": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.List{C},
				},
				"D": {
					Info:  testDAGStep([]string{"B", "C"}),
					Nodes: node.List{D},
				},
			}
			testflow.CreateInitialDAG(steps, rootNode)

			Expect(A.Children).To(HaveLen(2))
			Expect(A.Children).To(ConsistOf(B, C))

			Expect(B.Parents).To(HaveLen(1))
			Expect(B.Parents).To(ContainElement(A))

			Expect(C.Parents).To(HaveLen(1))
			Expect(C.Parents).To(ContainElement(A))

			Expect(D.Parents).To(HaveLen(2))
			Expect(D.Parents).To(ConsistOf(B, C))
		})

	})

	Context("serial steps", func() {
		It("should reorder a DAG with 1 parallel and 1 serial step", func() {
			A := testNode(nil, &defaultTestDef)
			Bs := testNode(node.List{A}, &serialTestDef)
			C := testNode(node.List{A}, &defaultTestDef)
			D := testNode(node.List{Bs, C}, &defaultTestDef)

			Expect(testflow.ReorderChildrenOfNodes(node.List{A})).To(BeNil())

			Expect(A.Children).To(HaveLen(1))
			Expect(A.Children[0]).To(Equal(C))

			Expect(C.Children).To(HaveLen(1))
			Expect(C.Children[0]).To(Equal(Bs))

			Expect(Bs.Children).To(HaveLen(1))
			Expect(Bs.Children[0]).To(Equal(D))

			Expect(D.Parents).To(HaveLen(1))
			Expect(D.Parents[0]).To(Equal(Bs))
		})

		It("should reorder a DAG with 2 parallel and 2 serial step", func() {
			A := testNode(nil, &defaultTestDef)
			Bs := testNode(node.List{A}, &serialTestDef)
			C := testNode(node.List{A}, &defaultTestDef)
			D := testNode(node.List{A}, &defaultTestDef)
			Es := testNode(node.List{A}, &serialTestDef)
			F := testNode(node.List{Bs, C, D, Es}, &defaultTestDef)

			Expect(testflow.ReorderChildrenOfNodes(node.List{A})).To(BeNil())

			Expect(A.Children).To(HaveLen(2))

			Expect(C.Children).To(HaveLen(1))
			Expect(C.Children[0]).To(Equal(Bs))
			Expect(D.Children).To(HaveLen(1))
			Expect(D.Children[0]).To(Equal(Bs))

			Expect(Bs.Children).To(HaveLen(1))
			Expect(Bs.Children[0]).To(Equal(Es))
			Expect(Es.Children).To(HaveLen(1))
			Expect(Es.Children[0]).To(Equal(F))

			Expect(F.Parents).To(HaveLen(1))
			Expect(F.Parents[0]).To(Equal(Es))
		})

		It("should change a reordered DAG", func() {
			A := testNode(nil, &defaultTestDef)
			Bs := testNode(node.List{A}, &serialTestDef)
			C := testNode(node.List{A}, &defaultTestDef)
			D := testNode(node.List{Bs, C}, &defaultTestDef)

			Expect(testflow.ReorderChildrenOfNodes(node.List{A})).To(BeNil())
			Expect(testflow.ReorderChildrenOfNodes(node.List{A})).To(BeNil())

			Expect(len(A.Children)).To(Equal(1))
			Expect(A.Children[0]).To(Equal(C))

			Expect(len(C.Children)).To(Equal(1))
			Expect(C.Children[0]).To(Equal(Bs))

			Expect(len(Bs.Children)).To(Equal(1))
			Expect(Bs.Children[0]).To(Equal(D))

			Expect(len(D.Parents)).To(Equal(1))
			Expect(D.Parents[0]).To(Equal(Bs))

		})
	})

	Context("Apply namespaces", func() {
		It("should set the last serial parent as artifact source", func() {
			rootNode := testNode(nil, &defaultTestDef)
			A := testNode(node.List{rootNode}, &defaultTestDef)
			B := testNode(node.List{A}, &defaultTestDef)
			C := testNode(node.List{A}, &defaultTestDef)
			D := testNode(node.List{B, C}, &defaultTestDef)
			steps := map[string]*testflow.Step{
				"A": {
					Info:  testDAGStep([]string{}),
					Nodes: node.List{A},
				},
				"B": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.List{B},
				},
				"C": {
					Info:  testDAGStep([]string{"A"}),
					Nodes: node.List{C},
				},
				"D": {
					Info:  testDAGStep([]string{"B", "C"}),
					Nodes: node.List{D},
				},
			}

			Expect(testflow.ApplyOutputNamespaces(steps)).ToNot(HaveOccurred())

			Expect(A.GetInputSource()).To(Equal(rootNode))

			Expect(B.GetInputSource()).To(Equal(A))
			Expect(C.GetInputSource()).To(Equal(A))
			Expect(D.GetInputSource()).To(Equal(A))
		})

	})

	Context("serial nodes", func() {
		It("should mark real serial nodes as serial", func() {
			A := testNode(nil, &defaultTestDef)
			B := testNode(node.List{A}, &defaultTestDef)
			C := testNode(node.List{A}, &defaultTestDef)
			D := testNode(node.List{B, C}, &defaultTestDef)

			testflow.SetSerialNodes(A)

			Expect(A.IsSerial()).To(BeFalse())
			Expect(B.IsSerial()).To(BeFalse())
			Expect(C.IsSerial()).To(BeFalse())
			Expect(D.IsSerial()).To(BeTrue())
		})

		It("should mark real serial nodes as serial in a huge DAG", func() {
			A := testNode(nil, &defaultTestDef)
			B := testNode(node.List{A}, &defaultTestDef)
			C := testNode(node.List{A}, &defaultTestDef)
			D := testNode(node.List{B}, &defaultTestDef)
			E := testNode(node.List{C}, &defaultTestDef)
			F := testNode(node.List{C}, &defaultTestDef)
			G := testNode(node.List{D, E, F}, &defaultTestDef)
			H := testNode(node.List{G}, &defaultTestDef)
			I := testNode(node.List{G}, &defaultTestDef)
			J := testNode(node.List{H, I}, &defaultTestDef)
			K := testNode(node.List{J}, &defaultTestDef)

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

func testNode(parents node.List, td *testdefinition.TestDefinition) *node.Node {
	n := &node.Node{
		Parents:        parents,
		TestDefinition: td,
	}

	for _, parent := range parents {
		parent.AddChildren(n)
	}

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
