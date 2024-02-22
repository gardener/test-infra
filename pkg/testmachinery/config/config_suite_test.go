// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config", func() {

	Context("Create", func() {

		DescribeTable("create config elements",
			func(configs []tmv1beta1.ConfigElement) {
				newConfigs := config.New(configs, config.LevelTestDefinition)
				Expect(len(configs)).To(Equal(len(newConfigs)))
				for i, newElem := range newConfigs {
					Expect(*newElem.Info).To(Equal(configs[i]))
					Expect(newElem.Name).ToNot(Equal(""))
				}
			},
			Entry("1 element", []tmv1beta1.ConfigElement{{Name: "test1", Value: "", Type: "env"}}),
			Entry("3 elements", []tmv1beta1.ConfigElement{{Name: "test1", Value: "", Type: "env"},
				{Name: "test2", Value: "", Type: "env"}, {Name: "test3", Value: "", Type: "env"}}),
			Entry("0 elements", []tmv1beta1.ConfigElement{}),
		)

	})

})
