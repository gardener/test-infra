// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"github.com/go-logr/logr"
	"github.com/onsi/ginkgo"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/test/resources"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
