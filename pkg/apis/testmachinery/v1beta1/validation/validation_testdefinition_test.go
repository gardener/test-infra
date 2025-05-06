// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1/validation"
)

var _ = Describe("TestDefinition Validation", func() {

	Context("validating testdefinitions", func() {
		var testdef *tmv1beta1.TestDefinition
		BeforeEach(func() {
			testdef = &tmv1beta1.TestDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testdefinitionname",
				},
				Spec: tmv1beta1.TestDefSpec{
					Command: []string{"bash"},
					Owner:   "test@corp.com",
				},
			}
		})

		It("should succeed when a name and a command is defined", func() {
			Expect(validation.ValidateTestDefinition(stdPath, testdef)).To(BeEmpty())
		})

		It("should succeed when a name contains '-'", func() {
			testdef.Name = "test-name"
			Expect(validation.ValidateTestDefinition(stdPath, testdef)).To(BeEmpty())
		})

		It("should fail when no name is defined", func() {
			testdef.Name = ""
			errList := validation.ValidateTestDefinition(stdPath, testdef)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("identifier.name"),
			}))))
		})

		It("should fail when the name contains a '.'", func() {
			testdef.Name = "test.name"
			errList := validation.ValidateTestDefinition(stdPath, testdef)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("identifier.name"),
			}))))
		})

		It("should fail when the name contains upper case letter", func() {
			testdef.Name = "testName"
			errList := validation.ValidateTestDefinition(stdPath, testdef)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("identifier.name"),
			}))))
		})

		It("should fail when no command is defined", func() {
			testdef.Spec.Command = []string{}
			errList := validation.ValidateTestDefinition(stdPath, testdef)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("identifier.spec.command"),
			}))))
		})

		It("should fail if no Owner is defined", func() {
			testdef.Spec.Owner = ""
			errList := validation.ValidateTestDefinition(stdPath, testdef)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("identifier.spec.owner"),
			}))))
		})

		It("should succeed when valid recipient is defined", func() {
			testdef.Spec.RecipientsOnFailure = []string{"test@corp.com"}
			errList := validation.ValidateTestDefinition(stdPath, testdef)
			Expect(errList).To(BeEmpty())
		})

		It("should succeed when valid recipient list is defined", func() {
			testdef.Spec.RecipientsOnFailure = []string{"test@corp.com", "test2@corp.com"}
			errList := validation.ValidateTestDefinition(stdPath, testdef)
			Expect(errList).To(BeEmpty())
		})

		It("should fail if any of recipient emails is invalid email", func() {
			testdef.Spec.RecipientsOnFailure = []string{"test@corp.com", "test2corp.com"}
			errList := validation.ValidateTestDefinition(stdPath, testdef)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("identifier.spec.recipientsOnFailure"),
			}))))
		})

		It("should fail when the recipient email is not a valid email", func() {
			testdef.Spec.RecipientsOnFailure = []string{"testcorp.com"}
			errList := validation.ValidateTestDefinition(stdPath, testdef)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("identifier.spec.recipientsOnFailure"),
			}))))
		})
	})
})
