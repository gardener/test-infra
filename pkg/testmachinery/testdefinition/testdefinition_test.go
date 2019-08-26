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

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
