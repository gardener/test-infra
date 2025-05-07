// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testflow_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	testutils "github.com/gardener/test-infra/test/utils"
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
			Expect(testflow.Validate(stdPath, tf, testutils.EmptyMockLocation, true)).To(BeEmpty())
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
			Expect(testflow.Validate(stdPath, tf, locations, true)).To(BeEmpty())
		})
	})
})
