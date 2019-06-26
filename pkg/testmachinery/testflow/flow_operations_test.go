package testflow_test

import (
	"testing"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
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
			rootNode := testNode("root", nil, defaultTestDef, nil)
			A := testNode("A", nil, defaultTestDef, testDAGStep([]string{}))
			B := testNode("B", nil, defaultTestDef, testDAGStep([]string{"A"}))
			steps := map[string]*testflow.Step{
				"A": {
					Info:  A.Step(),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  B.Step(),
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
			rootNode := testNode("root", nil, defaultTestDef, nil)
			A := testNode("A", nil, defaultTestDef, testDAGStep([]string{}))
			B := testNode("B", nil, defaultTestDef, testDAGStep([]string{"A"}))
			steps := map[string]*testflow.Step{
				"A": {
					Info:  A.Step(),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  B.Step(),
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
			rootNode := testNode("A", nil, defaultTestDef, nil)
			A := testNode("A", nil, defaultTestDef, testDAGStep([]string{}))
			B := testNode("B", nil, defaultTestDef, testDAGStep([]string{"A"}))
			C := testNode("C", nil, defaultTestDef, testDAGStep([]string{"A"}))
			D := testNode("D", nil, defaultTestDef, testDAGStep([]string{"B", "C"}))
			steps := map[string]*testflow.Step{
				"A": {
					Info:  A.Step(),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  B.Step(),
					Nodes: node.NewSet(B),
				},
				"C": {
					Info:  C.Step(),
					Nodes: node.NewSet(C),
				},
				"D": {
					Info:  D.Step(),
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
			A := testNode("A", nil, defaultTestDef, nil)
			Bs := testNode("Bs", node.NewSet(A), &serialTestDef, nil)
			C := testNode("C", node.NewSet(A), defaultTestDef, nil)
			D := testNode("D", node.NewSet(Bs, C), defaultTestDef, nil)

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
			A := testNode("A", nil, defaultTestDef, nil)
			Bs := testNode("Bs", node.NewSet(A), &serialTestDef, nil)
			C := testNode("C", node.NewSet(A), defaultTestDef, nil)
			D := testNode("D", node.NewSet(A), defaultTestDef, nil)
			Es := testNode("Es", node.NewSet(A), &serialTestDef, nil)
			F := testNode("F", node.NewSet(Bs, C, D, Es), defaultTestDef, nil)

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
			A := testNode("A", nil, defaultTestDef, nil)
			Bs := testNode("Bs", node.NewSet(A), &serialTestDef, nil)
			C := testNode("C", node.NewSet(A), defaultTestDef, nil)
			D := testNode("D", node.NewSet(Bs, C), defaultTestDef, nil)

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
			A := testNode("A", nil, defaultTestDef, nil)
			Bs := testNode("Bs", node.NewSet(A), &serialTestDef, nil)

			Expect(testflow.ReorderChildrenOfNodes(node.NewSet(A))).To(BeNil())

			Expect(A.Children).To(HaveLen(1))
			Expect(A.Children).To(HaveKey(Bs))

			Expect(Bs.Children).To(HaveLen(0))

		})
	})

	Context("Apply namespaces", func() {
		It("should set the last serial parent as artifact source", func() {
			rootNode := testNode("root", node.NewSet(), defaultTestDef, nil)
			A := testNode("A", node.NewSet(rootNode), defaultTestDef, testDAGStep([]string{}))
			B := testNode("B", node.NewSet(A), defaultTestDef, testDAGStep([]string{"A"}))
			C := testNode("C", node.NewSet(A), defaultTestDef, testDAGStep([]string{"A"}))
			D := testNode("D", node.NewSet(B, C), defaultTestDef, testDAGStep([]string{"B", "C"}))
			steps := map[string]*testflow.Step{
				"A": {
					Info:  A.Step(),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  B.Step(),
					Nodes: node.NewSet(B),
				},
				"C": {
					Info:  C.Step(),
					Nodes: node.NewSet(C),
				},
				"D": {
					Info:  D.Step(),
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

	Context("Apply config namespaces", func() {
		It("should propagate config from A to all nodes", func() {
			configOfA := []v1beta1.ConfigElement{
				{
					Type:  v1beta1.ConfigTypeEnv,
					Name:  "A",
					Value: "Aa",
				},
			}

			rootNode := testNode("root", node.NewSet(), testdefinition.NewEmpty(), nil)
			A := testNode("A", node.NewSet(rootNode), testdefinition.NewEmpty(), testDAGStepWithConfig([]string{}, configOfA))
			B := testNode("B", node.NewSet(A), testdefinition.NewEmpty(), testDAGStep([]string{"A"}))
			C := testNode("C", node.NewSet(A), testdefinition.NewEmpty(), testDAGStep([]string{"A"}))
			D := testNode("D", node.NewSet(B, C), testdefinition.NewEmpty(), testDAGStep([]string{"B", "B"}))

			steps := map[string]*testflow.Step{
				"A": {
					Info:  A.Step(),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  B.Step(),
					Nodes: node.NewSet(B),
				},
				"C": {
					Info:  C.Step(),
					Nodes: node.NewSet(C),
				},
				"D": {
					Info:  D.Step(),
					Nodes: node.NewSet(D),
				},
			}

			testflow.ApplyConfigScope(steps)

			Expect(B.TestDefinition.GetConfig()).To(HaveKey(configOfA[0].Name))
			Expect(C.TestDefinition.GetConfig()).To(HaveKey(configOfA[0].Name))
			Expect(D.TestDefinition.GetConfig()).To(HaveKey(configOfA[0].Name))
		})

		It("should overwrite config from A in B", func() {
			configOfA := v1beta1.ConfigElement{
				Type:  v1beta1.ConfigTypeEnv,
				Name:  "A",
				Value: "Aa",
			}
			configOfB := v1beta1.ConfigElement{
				Type:  v1beta1.ConfigTypeEnv,
				Name:  "A",
				Value: "Ab",
			}

			rootNode := testNode("root", node.NewSet(), testdefinition.NewEmpty(), nil)
			A := testNode("A", node.NewSet(rootNode), testdefinition.NewEmpty(), testDAGStepWithConfig([]string{}, []v1beta1.ConfigElement{configOfA}))
			B := testNode("B", node.NewSet(A), testdefinition.NewEmpty(), testDAGStepWithConfig([]string{"A"}, []v1beta1.ConfigElement{configOfB}))
			C := testNode("C", node.NewSet(A), testdefinition.NewEmpty(), testDAGStep([]string{"A"}))
			D := testNode("D", node.NewSet(B, C), testdefinition.NewEmpty(), testDAGStep([]string{"B", "C"}))

			steps := map[string]*testflow.Step{
				"A": {
					Info:  A.Step(),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  B.Step(),
					Nodes: node.NewSet(B),
				},
				"C": {
					Info:  C.Step(),
					Nodes: node.NewSet(C),
				},
				"D": {
					Info:  D.Step(),
					Nodes: node.NewSet(D),
				},
			}

			testflow.ApplyConfigScope(steps)

			Expect(B.TestDefinition.GetConfig()).To(HaveKey(configOfA.Name))
			Expect(*B.TestDefinition.GetConfig()["A"].Info).To(Equal(configOfB))

			Expect(C.TestDefinition.GetConfig()).To(HaveKey(configOfA.Name))
			Expect(*C.TestDefinition.GetConfig()["A"].Info).To(Equal(configOfA))
			Expect(D.TestDefinition.GetConfig()).To(HaveKey(configOfA.Name))
		})

		It("should overwrite config A from node A in B but keep config B", func() {
			configOfAa := v1beta1.ConfigElement{
				Type:  v1beta1.ConfigTypeEnv,
				Name:  "A",
				Value: "Aa",
			}
			configOfBa := v1beta1.ConfigElement{
				Type:  v1beta1.ConfigTypeEnv,
				Name:  "A",
				Value: "Ab",
			}
			configOfBb := v1beta1.ConfigElement{
				Type:  v1beta1.ConfigTypeEnv,
				Name:  "B",
				Value: "Bb",
			}

			rootNode := testNode("root", node.NewSet(), testdefinition.NewEmpty(), nil)
			A := testNode("A", node.NewSet(rootNode), testdefinition.NewEmpty(), testDAGStepWithConfig([]string{}, []v1beta1.ConfigElement{configOfAa}))
			B := testNode("B", node.NewSet(A), testdefinition.NewEmpty(), testDAGStepWithConfig([]string{"A"}, []v1beta1.ConfigElement{configOfBa, configOfBb}))
			C := testNode("C", node.NewSet(A), testdefinition.NewEmpty(), testDAGStep([]string{"A"}))
			D := testNode("D", node.NewSet(B, C), testdefinition.NewEmpty(), testDAGStep([]string{"B", "C"}))

			steps := map[string]*testflow.Step{
				"A": {
					Info:  A.Step(),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  B.Step(),
					Nodes: node.NewSet(B),
				},
				"C": {
					Info:  C.Step(),
					Nodes: node.NewSet(C),
				},
				"D": {
					Info:  D.Step(),
					Nodes: node.NewSet(D),
				},
			}

			testflow.ApplyConfigScope(steps)

			Expect(B.TestDefinition.GetConfig()).To(HaveKey(configOfAa.Name))
			Expect(*B.TestDefinition.GetConfig()["A"].Info).To(Equal(configOfBa))
			Expect(*B.TestDefinition.GetConfig()["B"].Info).To(Equal(configOfBb))

			Expect(C.TestDefinition.GetConfig()).To(HaveKey(configOfAa.Name))
			Expect(*C.TestDefinition.GetConfig()["A"].Info).To(Equal(configOfAa))
			Expect(D.TestDefinition.GetConfig()).To(HaveKey(configOfAa.Name))
		})

		It("should propagate config A from node A and config B from node B to C", func() {
			configOfAa := v1beta1.ConfigElement{
				Type:  v1beta1.ConfigTypeEnv,
				Name:  "A",
				Value: "Aa",
			}
			configOfBa := v1beta1.ConfigElement{
				Type:  v1beta1.ConfigTypeEnv,
				Name:  "A",
				Value: "Ab",
			}
			configOfBb := v1beta1.ConfigElement{
				Type:  v1beta1.ConfigTypeEnv,
				Name:  "B",
				Value: "Bp",
			}

			rootNode := testNode("root", node.NewSet(), testdefinition.NewEmpty(), nil)
			A := testNode("A", node.NewSet(rootNode), testdefinition.NewEmpty(), testDAGStepWithConfig([]string{}, []v1beta1.ConfigElement{configOfAa}))
			B := testNode("B", node.NewSet(A), testdefinition.NewEmpty(), testDAGStepWithConfig([]string{"A"}, []v1beta1.ConfigElement{configOfBa, configOfBb}))
			Bp := testNode("Bp", node.NewSet(A), testdefinition.NewEmpty(), testDAGStep([]string{"A"}))
			C := testNode("C", node.NewSet(B), testdefinition.NewEmpty(), testDAGStep([]string{"B"}))
			D := testNode("D", node.NewSet(Bp, C), testdefinition.NewEmpty(), testDAGStep([]string{"Bp", "C"}))

			steps := map[string]*testflow.Step{
				"A": {
					Info:  A.Step(),
					Nodes: node.NewSet(A),
				},
				"B": {
					Info:  B.Step(),
					Nodes: node.NewSet(B),
				},
				"Bp": {
					Info:  Bp.Step(),
					Nodes: node.NewSet(Bp),
				},
				"C": {
					Info:  C.Step(),
					Nodes: node.NewSet(C),
				},
				"D": {
					Info:  D.Step(),
					Nodes: node.NewSet(D),
				},
			}

			testflow.ApplyConfigScope(steps)

			Expect(B.TestDefinition.GetConfig()).To(HaveKey("A"))
			Expect(B.TestDefinition.GetConfig()).To(HaveKey("B"))
			Expect(*B.TestDefinition.GetConfig()["A"].Info).To(Equal(configOfBa))
			Expect(*B.TestDefinition.GetConfig()["B"].Info).To(Equal(configOfBb))

			Expect(Bp.TestDefinition.GetConfig()).To(HaveKey("A"))
			Expect(*Bp.TestDefinition.GetConfig()["A"].Info).To(Equal(configOfAa))

			Expect(C.TestDefinition.GetConfig()).To(HaveKey("A"))
			Expect(*C.TestDefinition.GetConfig()["A"].Info).To(Equal(configOfBa))

			Expect(C.TestDefinition.GetConfig()).To(HaveKey("B"))
			Expect(*C.TestDefinition.GetConfig()["B"].Info).To(Equal(configOfBb))

			Expect(D.TestDefinition.GetConfig()).To(HaveKey("A"))
		})

	})

	Context("serial nodes", func() {
		It("should mark real serial nodes as serial", func() {
			A := testNode("A", node.NewSet(), defaultTestDef, nil)
			B := testNode("B", node.NewSet(A), defaultTestDef, nil)
			C := testNode("C", node.NewSet(A), defaultTestDef, nil)
			D := testNode("D", node.NewSet(B, C), defaultTestDef, nil)

			testflow.SetSerialNodes(A)

			Expect(A.IsSerial()).To(BeFalse())
			Expect(B.IsSerial()).To(BeFalse())
			Expect(C.IsSerial()).To(BeFalse())
			Expect(D.IsSerial()).To(BeTrue())
		})

		It("should mark all nodes as serial", func() {
			A := testNode("A", node.NewSet(), defaultTestDef, nil)
			B := testNode("B", node.NewSet(A), defaultTestDef, nil)
			C := testNode("C", node.NewSet(B), defaultTestDef, nil)
			D := testNode("D", node.NewSet(C), defaultTestDef, nil)

			testflow.SetSerialNodes(A)

			Expect(A.IsSerial()).To(BeFalse())
			Expect(B.IsSerial()).To(BeTrue())
			Expect(C.IsSerial()).To(BeTrue())
			Expect(D.IsSerial()).To(BeTrue())
		})

		It("should mark real serial nodes as serial in a huge DAG", func() {
			A := testNode("A", node.NewSet(), testdefinition.NewEmpty(), nil)
			B := testNode("B", node.NewSet(A), testdefinition.NewEmpty(), nil)
			C := testNode("C", node.NewSet(A), testdefinition.NewEmpty(), nil)
			D := testNode("D", node.NewSet(B), testdefinition.NewEmpty(), nil)
			E := testNode("E", node.NewSet(C), testdefinition.NewEmpty(), nil)
			F := testNode("F", node.NewSet(C), testdefinition.NewEmpty(), nil)
			G := testNode("G", node.NewSet(D, E, F), testdefinition.NewEmpty(), nil)
			H := testNode("H", node.NewSet(G), testdefinition.NewEmpty(), nil)
			I := testNode("I", node.NewSet(G), testdefinition.NewEmpty(), nil)
			J := testNode("J", node.NewSet(I), testdefinition.NewEmpty(), nil)
			K := testNode("K", node.NewSet(H, J), testdefinition.NewEmpty(), nil)
			L := testNode("L", node.NewSet(K), testdefinition.NewEmpty(), nil)

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
			Expect(J.IsSerial()).To(BeFalse())
			Expect(K.IsSerial()).To(BeTrue())
			Expect(L.IsSerial()).To(BeTrue())
		})
	})
})

func testNode(name string, parents node.Set, td *testdefinition.TestDefinition, step *v1beta1.DAGStep) *node.Node {
	if parents == nil {
		parents = node.NewSet()
	}
	n := node.NewEmpty(name)
	n.Parents = parents
	n.Children = node.NewSet()
	n.TestDefinition = td

	n.SetStep(step)
	if td != nil {
		td.SetName(name)
		if step != nil {
			td.AddConfig(config.New(step.Definition.Config, config.LevelStep))
		}
	}

	for parent := range parents {
		parent.AddChildren(n)
	}

	return n
}

func testDAGStep(dependencies []string) *v1beta1.DAGStep {
	return &v1beta1.DAGStep{
		DependsOn: dependencies,
		Definition: v1beta1.StepDefinition{
			Config: make([]v1beta1.ConfigElement, 0),
		},
	}
}

func testDAGStepWithConfig(dependencies []string, elements []v1beta1.ConfigElement) *v1beta1.DAGStep {
	step := testDAGStep(dependencies)
	step.Definition.Config = elements

	return step
}

var serialTestDef = func() testdefinition.TestDefinition {
	return testdefinition.TestDefinition{
		Info: &v1beta1.TestDefinition{
			Spec: v1beta1.TestDefSpec{
				Behavior: []string{"serial"},
			},
		},
		Template: &argov1.Template{},
	}
}()
var defaultTestDef = testdefinition.NewEmpty()

func testDefWithConfig(cfgs []v1beta1.ConfigElement) *testdefinition.TestDefinition {
	td := testdefinition.NewEmpty()
	td.AddConfig(config.New(cfgs, config.LevelTestDefinition))
	return td
}
