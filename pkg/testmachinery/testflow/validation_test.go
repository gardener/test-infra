// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testflow", func() {
	Context("validatation", func() {
		It("should fail when no testdefs are found", func() {
			tf := tmv1beta1.TestFlow{}
			locations := &testDefinitionsMock{}
			Expect(testflow.Validate("identifier", tf, locations, false)).To(HaveOccurred())
		})

		It("should fail when step names are not unique", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
			}
			Expect(testflow.Validate("identifier", tf, defaultMockLocation, true)).To(HaveOccurred())
		})

		It("should fail when dependent steps do not exist", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
				{
					Name:      "int-test2",
					DependsOn: []string{"bla"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
			}
			Expect(testflow.Validate("identifier", tf, defaultMockLocation, true)).To(HaveOccurred())
		})

		It("should fail when dependencies have a cycle", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
				{
					Name:      "int-test1",
					DependsOn: []string{"int-test3", "int-test"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
				{
					Name:      "int-test2",
					DependsOn: []string{"int-test1"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
				{
					Name:      "int-test3",
					DependsOn: []string{"int-test1"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
			}
			Expect(testflow.Validate("identifier", tf, defaultMockLocation, true)).To(HaveOccurred())
		})

		It("should fail when artifactsFrom and useGlobalArtifacts are used at the same time", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
				{
					Name:      "int-test2",
					DependsOn: []string{"int-test"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
					ArtifactsFrom:      "int-test",
					UseGlobalArtifacts: true,
				},
			}
			Expect(testflow.Validate("identifier", tf, defaultMockLocation, true)).To(HaveOccurred())
		})

		It("should fail when step with artifactsFrom name does not exist", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
				{
					Name:      "int-test2",
					DependsOn: []string{"int-test"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
					ArtifactsFrom: "notExistingStepName",
				},
			}
			Expect(testflow.Validate("identifier", tf, defaultMockLocation, true)).To(HaveOccurred())
		})

		It("should fail when step with artifactsFrom name does not exist", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
				{
					Name:      "int-test2",
					DependsOn: []string{"int-test"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
					ArtifactsFrom: "int-test",
				},
				{
					Name:      "int-test3",
					DependsOn: []string{"int-test2"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
					ArtifactsFrom: "int-test2",
				},
				{
					Name:      "int-test4",
					DependsOn: []string{"int-test2"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
					ArtifactsFrom: "int-test",
				},
			}
			Expect(testflow.Validate("identifier", tf, defaultMockLocation, true)).To(Succeed())
		})

		It("should fail artifactsFrom step name represents a succeeding step", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
				{
					Name:      "int-test2",
					DependsOn: []string{"int-test"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
					ArtifactsFrom: "int-test",
				},
				{
					Name:      "int-test3",
					DependsOn: []string{"int-test2"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
					ArtifactsFrom: "int-test4",
				},
				{
					Name:      "int-test4",
					DependsOn: []string{"int-test2"},
					Definition: tmv1beta1.StepDefinition{
						Name: "testdefname",
					},
				},
			}
			Expect(testflow.Validate("identifier", tf, defaultMockLocation, true)).To(HaveOccurred())
		})

		It("should fail when labels without matching testdefs are found", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "noMatchingLabel",
					},
				},
			}
			Expect(testflow.Validate("identifier", tf, defaultMockLocation, false)).To(HaveOccurred())
		})

		It("should succeed when an empty flow is ignored", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Label: "testdefname",
					},
				},
			}
			Expect(testflow.Validate("identifier", tf, defaultMockLocation, true)).ToNot(HaveOccurred())
		})

		It("should succeed when a testdef can be found", func() {
			tf := tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Label: "noMatchingLabel",
					},
				},
			}
			locations := &testDefinitionsMock{
				getTestDefinitions: func(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error) {
					testdefs := []*testdefinition.TestDefinition{
						{
							Location: &locationMock{},
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
			Expect(testflow.Validate("identifier", tf, locations, true)).ToNot(HaveOccurred())
		})
	})
})

type testDefinitionsMock struct {
	getTestDefinitions func(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error)
}

func (t *testDefinitionsMock) GetTestDefinitions(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error) {
	return t.getTestDefinitions(step)
}

type locationMock struct {
}

func (l *locationMock) Name() string {
	return "locationmock"
}

func (l *locationMock) Type() tmv1beta1.LocationType {
	return "mock"
}

func (l *locationMock) SetTestDefs(_ map[string]*testdefinition.TestDefinition) error {
	return nil
}

func (l *locationMock) GetLocation() *tmv1beta1.TestLocation {
	return nil
}

var defaultMockLocation = &testDefinitionsMock{
	getTestDefinitions: func(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error) {
		testdefs := []*testdefinition.TestDefinition{
			{
				Location: &locationMock{},
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
