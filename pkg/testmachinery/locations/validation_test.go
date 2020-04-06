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

package locations_test

import (
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"testing"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testdefinition Suite")
}

var _ = Describe("TestDefinition Validation", func() {

	Context("validating testdefinition locations", func() {
		var location tmv1beta1.TestLocation
		BeforeEach(func() {
			location = tmv1beta1.TestLocation{
				Type:     "git",
				Repo:     "testrepo",
				Revision: "master",
				HostPath: "/home/testing",
			}
			testmachinery.GetConfig().TestMachineryConfiguration.Insecure = false
		})

		It("should fail when no type is defined", func() {
			location.Type = ""
			Expect(locations.ValidateTestLocation("identifier", location)).To(HaveOccurred())
		})
		It("should fail when an unknown type is defined", func() {
			location.Type = "unknownType"
			Expect(locations.ValidateTestLocation("identifier", location)).To(HaveOccurred())
		})

		Context("when location type is git", func() {
			It("should succeed when repo and revision are defined", func() {
				Expect(locations.ValidateTestLocation("identifier", location)).ToNot(HaveOccurred())
			})
		})

		Context("when location type is local", func() {
			It("should succeed when hostPath is defined and the application running in insecure mode", func() {
				location.Type = "local"
				testmachinery.GetConfig().TestMachineryConfiguration.Insecure = true
				Expect(locations.ValidateTestLocation("identifier", location)).ToNot(HaveOccurred())
			})

			It("should fail when hostPath is not specified", func() {
				location.Type = "local"
				location.HostPath = ""
				Expect(locations.ValidateTestLocation("identifier", location)).To(HaveOccurred())
			})

			It("should fail when the testmachinery is running in secure mode", func() {
				location.Type = "local"
				Expect(locations.ValidateTestLocation("identifier", location)).To(HaveOccurred())
			})
		})
	})
})
