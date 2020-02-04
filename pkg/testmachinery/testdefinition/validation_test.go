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

package testdefinition_test

import (
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
			Expect(testdefinition.Validate("identifier", testdef)).ToNot(HaveOccurred())
		})

		It("should succeed when a name contains '-'", func() {
			testdef.Name = "test-name"
			Expect(testdefinition.Validate("identifier", testdef)).ToNot(HaveOccurred())
		})

		It("should fail when no name is defined", func() {
			testdef.Name = ""
			Expect(testdefinition.Validate("identifier", testdef)).To(HaveOccurred())
		})

		It("should fail when the name contains a '.'", func() {
			testdef.Name = "test.name"
			Expect(testdefinition.Validate("identifier", testdef)).To(HaveOccurred())
		})

		It("should fail when the name contains upper case letter", func() {
			testdef.Name = "testName"
			Expect(testdefinition.Validate("identifier", testdef)).To(HaveOccurred())
		})

		It("should fail when no command is defined", func() {
			testdef.Spec.Command = []string{}
			Expect(testdefinition.Validate("identifier", testdef)).To(HaveOccurred())
		})

		It("should fail if no Owner is defined", func() {
			testdef.Spec.Owner = ""
			Expect(testdefinition.Validate("identifier", testdef)).To(HaveOccurred())
		})

		It("should succeed when valid recipient is defined", func() {
			testdef.Spec.RecipientsOnFailure = []string{"test@corp.com"}
			Expect(testdefinition.Validate("identifier", testdef)).ToNot(HaveOccurred())
		})

		It("should succeed when valid recipient list is defined", func() {
			testdef.Spec.RecipientsOnFailure = []string{"test@corp.com", "test2@corp.com"}
			Expect(testdefinition.Validate("identifier", testdef)).ToNot(HaveOccurred())
		})

		It("should fail if any of recipient emails is invalid email", func() {
			testdef.Spec.RecipientsOnFailure = []string{"test@corp.com", "test2corp.com"}
			Expect(testdefinition.Validate("identifier", testdef)).To(HaveOccurred())
		})

		It("should fail when the recipient email is not a valid email", func() {
			testdef.Spec.RecipientsOnFailure = []string{"testcorp.com"}
			Expect(testdefinition.Validate("identifier", testdef)).To(HaveOccurred())
		})
	})
})
