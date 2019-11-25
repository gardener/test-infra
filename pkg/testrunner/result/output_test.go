package result_test

import (
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	"github.com/gardener/test-infra/test/resources"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestValidationWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Result collection Integration Test Suite")
}

var _ = Describe("output generation tests", func() {
	log := logger.Log.WithName("output_test")
	Context("configuration", func() {
		It("should include environment variable configuration to in the metadata", func() {
			configElement := v1beta1.ConfigElement{
				Type:  v1beta1.ConfigTypeEnv,
				Name:  "test",
				Value: "val",
			}
			tr := resources.GetBasicTestrun("", "")

			tr.Status = v1beta1.TestrunStatus{
				Steps: []*v1beta1.StepStatus{
					{
						TestDefinition: v1beta1.StepStatusTestDefinition{
							Config: []*v1beta1.ConfigElement{
								&configElement,
							},
						},
					},
				},
			}

			_, summaries, err := result.DetermineTestrunSummary(tr, &testrunner.Metadata{}, &result.Config{}, nil, log)
			Expect(err).ToNot(HaveOccurred())

			Expect(summaries).To(HaveLen(1))
			Expect(summaries[0].Metadata.Configuration).To(HaveLen(1))
			Expect(summaries[0].Metadata.Configuration).To(HaveKey(configElement.Name))
			Expect(summaries[0].Metadata.Configuration[configElement.Name]).To(Equal(configElement.Value))
		})
	})

	It("should exnclude file variable configuration from in the metadata", func() {
		configElement := v1beta1.ConfigElement{
			Type:  v1beta1.ConfigTypeFile,
			Name:  "test",
			Value: "val",
		}
		tr := resources.GetBasicTestrun("", "")

		tr.Status = v1beta1.TestrunStatus{
			Steps: []*v1beta1.StepStatus{
				{
					TestDefinition: v1beta1.StepStatusTestDefinition{
						Config: []*v1beta1.ConfigElement{
							&configElement,
						},
					},
				},
			},
		}
		_, summaries, err := result.DetermineTestrunSummary(tr, &testrunner.Metadata{}, &result.Config{}, nil, log)
		Expect(err).ToNot(HaveOccurred())

		Expect(summaries).To(HaveLen(1))
		Expect(summaries[0].Metadata.Configuration).To(HaveLen(0))
	})
})
