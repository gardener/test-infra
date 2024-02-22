// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"github.com/go-logr/logr"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/test/resources"
)

var _ = Describe("output generation tests", func() {
	var (
		c *collector
	)

	ginkgo.BeforeEach(func() {
		c = &collector{
			log: logr.Discard(),
		}
	})

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

			_, summaries, err := c.generateSummary(tr, &metadata.Metadata{})
			Expect(err).ToNot(HaveOccurred())

			Expect(summaries).To(HaveLen(1))
			Expect(summaries[0].Metadata.Configuration).To(HaveLen(1))
			Expect(summaries[0].Metadata.Configuration).To(HaveKey(configElement.Name))
			Expect(summaries[0].Metadata.Configuration[configElement.Name]).To(Equal(configElement.Value))
		})
	})

	It("should exclude file variable configuration from in the metadata", func() {
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
		_, summaries, err := c.generateSummary(tr, &metadata.Metadata{})
		Expect(err).ToNot(HaveOccurred())

		Expect(summaries).To(HaveLen(1))
		Expect(summaries[0].Metadata.Configuration).To(HaveLen(0))
	})
})
