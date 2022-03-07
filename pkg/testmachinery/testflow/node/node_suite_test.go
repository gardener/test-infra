package node

import (
	"testing"

	apiv1 "k8s.io/api/core/v1"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

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

	Context("ProjectedTokenMounts", func() {

		var node Node

		BeforeEach(func() {
			node = Node{
				TestDefinition: &testdefinition.TestDefinition{
					Template: &argov1.Template{
						Container: &apiv1.Container{},
					},
				},
			}
		})

		It("should add volumes for one ProjectedTokenMount", func() {
			projectedTokenMounts := []ProjectedTokenMount{
				{
					Audience:          "test-audience",
					ExpirationSeconds: 60,
					Name:              "test-name",
					MountPath:         "/test/path",
				},
			}

			node.addProjectedToken(projectedTokenMounts)

			Expect(len(node.TestDefinition.Template.Volumes)).To(Equal(1))
			Expect(len(node.TestDefinition.Template.Container.VolumeMounts)).To(Equal(1))

			Expect(node.TestDefinition.Template.Volumes[0].Name).To(Equal("token-0"))
			Expect(node.TestDefinition.Template.Volumes[0].Projected.Sources[0].ServiceAccountToken.Path).To(Equal("test-name"))
			Expect(*(node.TestDefinition.Template.Volumes[0].Projected.Sources[0].ServiceAccountToken.ExpirationSeconds)).To(Equal(int64(60)))
			Expect(node.TestDefinition.Template.Volumes[0].Projected.Sources[0].ServiceAccountToken.Audience).To(Equal("test-audience"))

			Expect(node.TestDefinition.Template.Container.VolumeMounts[0].MountPath).To(Equal("/test/path"))
		})

		It("should add volumes for several ProjectedTokenMount", func() {
			projectedTokenMounts := []ProjectedTokenMount{
				{
					Audience:          "first-audience",
					ExpirationSeconds: 100,
					Name:              "first-name",
					MountPath:         "/first/path",
				},
				{
					Audience:          "second-audience",
					ExpirationSeconds: 200,
					Name:              "second-name",
					MountPath:         "/second/path",
				},
			}
			node.addProjectedToken(projectedTokenMounts)

			Expect(len(node.TestDefinition.Template.Volumes)).To(Equal(2))
			Expect(len(node.TestDefinition.Template.Container.VolumeMounts)).To(Equal(2))
		})

		It("should add another Source to an existing volume", func() {
			projectedTokenMounts := []ProjectedTokenMount{
				{
					Audience:          "first-audience",
					ExpirationSeconds: 100,
					Name:              "first-name",
					MountPath:         "/same/path",
				},
				{
					Audience:          "second-audience",
					ExpirationSeconds: 200,
					Name:              "second-name",
					MountPath:         "/same/path",
				},
			}
			node.addProjectedToken(projectedTokenMounts)

			Expect(len(node.TestDefinition.Template.Volumes)).To(Equal(1))
			Expect(len(node.TestDefinition.Template.Volumes[0].Projected.Sources)).To(Equal(2))
			Expect(len(node.TestDefinition.Template.Container.VolumeMounts)).To(Equal(1))

			Expect(node.TestDefinition.Template.Volumes[0].Name).To(Equal("token-0"))
			Expect(node.TestDefinition.Template.Volumes[0].Projected.Sources[0].ServiceAccountToken.Path).To(Equal("first-name"))
			Expect(node.TestDefinition.Template.Volumes[0].Projected.Sources[1].ServiceAccountToken.Path).To(Equal("second-name"))
			Expect(*(node.TestDefinition.Template.Volumes[0].Projected.Sources[0].ServiceAccountToken.ExpirationSeconds)).To(Equal(int64(100)))
			Expect(*(node.TestDefinition.Template.Volumes[0].Projected.Sources[1].ServiceAccountToken.ExpirationSeconds)).To(Equal(int64(200)))
			Expect(node.TestDefinition.Template.Volumes[0].Projected.Sources[0].ServiceAccountToken.Audience).To(Equal("first-audience"))
			Expect(node.TestDefinition.Template.Volumes[0].Projected.Sources[1].ServiceAccountToken.Audience).To(Equal("second-audience"))

			Expect(node.TestDefinition.Template.Container.VolumeMounts[0].MountPath).To(Equal("/same/path"))

		})

		It("should add no volume, if there is nothing to do", func() {
			projectedTokenMounts := make([]ProjectedTokenMount, 0)
			node.addProjectedToken(projectedTokenMounts)

			Expect(len(node.TestDefinition.Template.Volumes)).To(BeZero())
			Expect(len(node.TestDefinition.Template.Container.VolumeMounts)).To(BeZero())
		})
	})
})
