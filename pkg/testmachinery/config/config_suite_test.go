// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package config_test

import (
	"testing"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config", func() {

	Context("Create", func() {

		DescribeTable("create config elements",
			func(configs []tmv1beta1.ConfigElement) {
				new := config.New(configs)
				Expect(len(configs)).To(Equal(len(new)))
				for i, newElem := range new {
					Expect(*newElem.Info).To(Equal(configs[i]))
					Expect(newElem.Name).ToNot(Equal(""))
				}
			},
			Entry("1 element", []tmv1beta1.ConfigElement{tmv1beta1.ConfigElement{Name: "test1", Value: "", Type: "env"}}),
			Entry("3 elements", []tmv1beta1.ConfigElement{tmv1beta1.ConfigElement{Name: "test1", Value: "", Type: "env"},
				tmv1beta1.ConfigElement{Name: "test2", Value: "", Type: "env"}, tmv1beta1.ConfigElement{Name: "test3", Value: "", Type: "env"}}),
			Entry("0 elements", []tmv1beta1.ConfigElement{}),
		)

	})

	Context("Validating config elements", func() {

		It("should fail without a config name", func() {
			elem := tmv1beta1.ConfigElement{
				Type: "en",
			}
			Expect(config.Validate("identifier", elem)).To(HaveOccurred())
		})

		It("should fail with unknown config type", func() {
			elem := tmv1beta1.ConfigElement{
				Name: "testConfig",
				Type: "en",
			}
			Expect(config.Validate("identifier", elem)).To(HaveOccurred())
		})

		It("should succeed with known config type", func() {
			elem := tmv1beta1.ConfigElement{
				Name: "testConfig",
				Type: "env",
			}
			Expect(config.Validate("identifier", elem)).ToNot(HaveOccurred())
		})
	})
})
