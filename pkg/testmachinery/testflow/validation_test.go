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

package testflow_test

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	testutils "github.com/gardener/test-infra/test/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var stdPath = field.NewPath("identifier")

var _ = Describe("Testflow", func() {
	Context("validatation", func() {
		It("should fail when no testdefs are found", func() {
			tf := tmv1beta1.TestFlow{}
			errList, _ := testflow.Validate(stdPath, tf, testutils.EmptyMockLocation, false)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("identifier"),
			}))))
		})

		It("should fail when a test step specifies a label with non existent testdefinitions", func() {
			tf := tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Label: "somelabel",
					},
				},
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
			}
			errList, _ := testflow.Validate(stdPath, tf, testutils.EmptyMockLocation, false)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("identifier[0].definition"),
			}))))
		})

		It("should fail when labels without matching testdefs are found", func() {
			tf := tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "noMatchingLabel",
					},
				},
			}
			errList, _ := testflow.Validate(stdPath, tf, testutils.EmptyMockLocation, false)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("identifier[0].definition"),
			}))))
		})

		It("should succeed when an empty flow is ignored", func() {
			tf := tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Label: "testdefname",
					},
				},
			}
			Expect(testflow.Validate(stdPath, tf, testutils.EmptyMockLocation, true)).To(HaveLen(0))
		})

		It("should succeed when a testdef can be found", func() {
			tf := tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Label: "noMatchingLabel",
					},
				},
			}
			locations := &testutils.LocationsMock{
				GetTestDefinitionsFunc: func(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error) {
					testdefs := []*testdefinition.TestDefinition{
						{
							Location: &testutils.TDLocationMock{},
							FileName: "file.name",
							Info: &tmv1beta1.TestDefinition{
								ObjectMeta: metav1.ObjectMeta{Name: "testdefname"},
								Spec:       tmv1beta1.TestDefSpec{Command: []string{"bash"}, Owner: "user@company.com"},
							},
						},
					}
					return testdefs, nil
				},
			}
			Expect(testflow.Validate(stdPath, tf, locations, true)).To(HaveLen(0))
		})
	})
})
