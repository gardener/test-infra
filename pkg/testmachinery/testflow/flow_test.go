package testflow_test

import (
	"fmt"

	"github.com/gardener/test-infra/pkg/testmachinery/testflow/node"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	testutils "github.com/gardener/test-infra/test/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("flow", func() {
	Context("init", func() {
		It("should set the root node as parent of all steps with no dependencies", func() {
			rootNode := testNode("root", nil, defaultTestDef, nil)
			locs := &testutils.LocationsMock{
				StepToTestDefinitions: map[string][]*testdefinition.TestDefinition{
					"create":        {testutils.TestDef("create")},
					"delete":        {testutils.TestDef("delete")},
					"tests-beta":    {testutils.TestDef("default"), testutils.SerialTestDef("serial"), testutils.DisruptiveTestDef("disruptive")},
					"tests-release": {testutils.TestDef("default"), testutils.SerialTestDef("serial"), testutils.DisruptiveTestDef("disruptive")},
				},
			}
			tf := tmv1beta1.TestFlow{}
			Expect(testutils.ReadYAMLFile("./testdata/testflow-00.yaml", &tf)).To(Succeed())

			_, err := testflow.NewFlow("testid", rootNode, tf, locs, nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(rootNode.Children.Len()).To(Equal(1), "the root node should have one create child")
			createNode := rootNode.Children.List()[0]
			Expect(createNode.TestDefinition.Info.Name).To(Equal("create"))

			// beta step
			Expect(createNode.Children.Len()).To(Equal(1), "the create node should have 1 test child")
			defaultBetaNode := createNode.Children.List()[0]
			Expect(defaultBetaNode.TestDefinition.Info.Name).To(Equal("default"))
			Expect(defaultBetaNode.Parents.Len()).To(Equal(1), "the default beta node should have one parent")
			Expect(defaultBetaNode.Parents.List()[0].TestDefinition.Info.Name).To(Equal("create"))

			Expect(defaultBetaNode.Children.Len()).To(Equal(1), "the disruptive beta node should have 1 test child")
			serialBetaNode := defaultBetaNode.Children.List()[0]
			Expect(serialBetaNode.TestDefinition.Info.Name).To(Equal("serial"))

			Expect(serialBetaNode.Children.Len()).To(Equal(1), "the default beta node should have 1 test child")
			disruptiveBetaNode := serialBetaNode.Children.List()[0]
			Expect(disruptiveBetaNode.TestDefinition.Info.Name).To(Equal("disruptive"))

			// release step
			Expect(disruptiveBetaNode.Children.Len()).To(Equal(1), "the disruptive beta node should have 1 test child")
			defaultReleaseNode := disruptiveBetaNode.Children.List()[0]
			Expect(defaultReleaseNode.TestDefinition.Info.Name).To(Equal("default"))

			Expect(defaultReleaseNode.Children.Len()).To(Equal(1), "the default release node should have 1 test child")
			serialReleaseNode := defaultReleaseNode.Children.List()[0]
			Expect(serialReleaseNode.TestDefinition.Info.Name).To(Equal("serial"))

			Expect(serialReleaseNode.Children.Len()).To(Equal(1), "the serial release node should have 1 test child")
			disruptiveReleaseNode := serialReleaseNode.Children.List()[0]
			Expect(disruptiveReleaseNode.TestDefinition.Info.Name).To(Equal("disruptive"))

			// delete
			Expect(disruptiveReleaseNode.Children.Len()).To(Equal(1), "the disruptive release node should have 1 test child")
			deleteNode := disruptiveReleaseNode.Children.List()[0]
			Expect(deleteNode.TestDefinition.Info.Name).To(Equal("delete"))
		})

		It("should set the root node as parent of all steps with no dependencies", func() {
			rootNode := testNode("root", nil, defaultTestDef, &tmv1beta1.DAGStep{})
			locs := &testutils.LocationsMock{
				StepToTestDefinitions: map[string][]*testdefinition.TestDefinition{
					"create":        {testutils.TestDef("create")},
					"delete":        {testutils.TestDef("delete")},
					"tests-beta":    {testutils.TestDef("default"), testutils.SerialTestDef("serial"), testutils.DisruptiveTestDef("disruptive")},
					"tests-release": {testutils.TestDef("default"), testutils.SerialTestDef("serial"), testutils.DisruptiveTestDef("disruptive")},
				},
			}
			tf := tmv1beta1.TestFlow{}
			Expect(testutils.ReadYAMLFile("./testdata/testflow-00.yaml", &tf)).To(Succeed())

			flow, err := testflow.NewFlow("testid", rootNode, tf, locs, nil)
			Expect(err).ToNot(HaveOccurred())

			trustedProjectedMounts := make([]node.ProjectedTokenMount, 0)
			unttrustedProjectedMounts := make([]node.ProjectedTokenMount, 0)
			tmpl := flow.GetDAGTemplate(testmachinery.PhaseRunning, trustedProjectedMounts, unttrustedProjectedMounts)

			for _, task := range tmpl.Tasks {
				if task.Name != "root" {
					Expect(len(task.Dependencies) > 0).To(BeTrue(), fmt.Sprintf("task %q should have at least one dependency", task.Name))
				}
			}
		})
	})
})
