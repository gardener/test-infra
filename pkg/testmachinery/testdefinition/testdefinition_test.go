// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testdefinition_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

var _ = Describe("TestDefinition", func() {

	Context("has label", func() {
		var testdef *testdefinition.TestDefinition
		BeforeEach(func() {
			testdef = &testdefinition.TestDefinition{
				Info: &tmv1beta1.TestDefinition{
					Spec: tmv1beta1.TestDefSpec{
						Labels: []string{"default", "default2", "not"},
					},
				},
			}
		})

		It("should return true when the labels match", func() {
			Expect(testdef.HasLabel("default")).To(BeTrue())
		})

		It("should return true when 2 labels match ", func() {
			Expect(testdef.HasLabel("default,default2")).To(BeTrue())
		})

		It("should return false when 1 labels does not match ", func() {
			Expect(testdef.HasLabel("default,default1")).To(BeFalse())
		})

		It("should return false when the testdefinition has 1 excluding label", func() {
			Expect(testdef.HasLabel("default,!not")).To(BeFalse())
		})

	})
})
