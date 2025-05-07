// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1/validation"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

var stdPath = field.NewPath("identifier")

var _ = Describe("Locations Validation", func() {

	Context("validating testdefinition locations", func() {
		var location tmv1beta1.TestLocation
		BeforeEach(func() {
			location = tmv1beta1.TestLocation{
				Type:     "git",
				Repo:     "testrepo",
				Revision: "master",
				HostPath: "/home/testing",
			}
			testmachinery.GetConfig().TestMachinery.Insecure = false
		})

		It("should fail when no type is defined", func() {
			location.Type = ""
			errList := validation.ValidateTestLocation(stdPath, location)
			Expect(errList).To(HaveLen(1))
		})
		It("should fail when an unknown type is defined", func() {
			location.Type = "unknownType"
			errList := validation.ValidateTestLocation(stdPath, location)
			Expect(errList).To(HaveLen(1))
		})

		Context("when location type is git", func() {
			It("should succeed when repo and revision are defined", func() {
				errList := validation.ValidateTestLocation(stdPath, location)
				Expect(errList).To(BeEmpty())
			})
		})

		Context("when location type is local", func() {
			It("should succeed when hostPath is defined and the application running in insecure mode", func() {
				location.Type = "local"
				testmachinery.GetConfig().TestMachinery.Insecure = true
				errList := validation.ValidateTestLocation(stdPath, location)
				Expect(errList).To(BeEmpty())
			})

			It("should fail when hostPath is not specified", func() {
				location.Type = "local"
				location.HostPath = ""
				errList := validation.ValidateTestLocation(stdPath, location)
				Expect(errList).To(HaveLen(1))
			})

			It("should fail when the testmachinery is running in secure mode", func() {
				location.Type = "local"
				errList := validation.ValidateTestLocation(stdPath, location)
				Expect(errList).To(HaveLen(1))
			})
		})
	})
})
