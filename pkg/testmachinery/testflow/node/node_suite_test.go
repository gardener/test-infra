package node

import (
	"testing"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	testutils "github.com/gardener/test-infra/test/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testflow Node Suite")
}

var _ = Describe("node operations", func() {
	Context("CreateNodesFromStep", func() {
		It("should set continueOnError to false for disruptive nodes", func() {
			step := &tmv1beta1.DAGStep{}
			step.Definition.ContinueOnError = true
			locs := &testutils.LocationsMock{
				TestDefinitions: []*testdefinition.TestDefinition{
					testutils.TestDef("default"),
					testutils.SerialTestDef("serial"),
					testutils.DisruptiveTestDef("disruptive"),
				},
			}
			nodes, err := CreateNodesFromStep(step, locs, nil, "")
			Expect(err).ToNot(HaveOccurred())
			nodes.List()[0].Step()

			Expect(nodes.Len()).To(Equal(3))
			// default
			Expect(nodes.List()[0].Step().Definition.ContinueOnError).To(BeTrue())
			// serial
			Expect(nodes.List()[1].Step().Definition.ContinueOnError).To(BeTrue())
			// disruptive
			Expect(nodes.List()[2].Step().Definition.ContinueOnError).To(BeFalse())

		})

	})
})
