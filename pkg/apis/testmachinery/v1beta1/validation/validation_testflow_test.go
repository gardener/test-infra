// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1/validation"
)

var _ = Describe("testflow", func() {

	It("should fail when step names are not unique", func() {
		tf := tmv1beta1.TestFlow{
			&tmv1beta1.DAGStep{
				Name: "int-test",
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name: "int-test",
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
		}
		errList := validation.ValidateTestFlow(stdPath, tf)
		Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeDuplicate),
			"Field": Equal("identifier[1].name"),
		}))))
	})

	It("should fail when dependent steps do not exist", func() {
		tf := tmv1beta1.TestFlow{
			&tmv1beta1.DAGStep{
				Name: "int-test",
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test2",
				DependsOn: []string{"bla"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
		}
		errList := validation.ValidateTestFlow(stdPath, tf)
		Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeNotFound),
			"Field": Equal("identifier[1].dependsOn"),
		}))))
	})

	It("should not fail when direct dependent steps does exist", func() {
		tf := tmv1beta1.TestFlow{
			&tmv1beta1.DAGStep{
				Name: "int-test",
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test2",
				DependsOn: []string{"int-test"},
				Definition: tmv1beta1.StepDefinition{
					Name: "int-test",
				},
			},
		}
		errList := validation.ValidateTestFlow(stdPath, tf)
		Expect(errList).To(HaveLen(0))
	})

	It("should fail when dependencies have a cycle", func() {
		tf := tmv1beta1.TestFlow{
			&tmv1beta1.DAGStep{
				Name: "int-test",
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test1",
				DependsOn: []string{"int-test3", "int-test"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test2",
				DependsOn: []string{"int-test1"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test3",
				DependsOn: []string{"int-test1"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
		}
		errList := validation.ValidateTestFlow(stdPath, tf)
		Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeForbidden),
			"Field": Equal("identifier[1].dependsOn"),
		}))))
	})

	It("should fail when artifactsFrom and useGlobalArtifacts are used at the same time", func() {
		tf := tmv1beta1.TestFlow{
			&tmv1beta1.DAGStep{
				Name: "int-test",
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test2",
				DependsOn: []string{"int-test"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
				ArtifactsFrom:      "int-test",
				UseGlobalArtifacts: true,
			},
		}
		errList := validation.ValidateTestFlow(stdPath, tf)
		Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeForbidden),
			"Field": Equal("identifier[1].useGlobalArtifacts"),
		}))))
	})

	It("should fail when step with artifactsFrom name does not exist", func() {
		tf := tmv1beta1.TestFlow{
			&tmv1beta1.DAGStep{
				Name: "int-test",
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test2",
				DependsOn: []string{"int-test"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
				ArtifactsFrom: "notExistingStepName",
			},
		}
		errList := validation.ValidateTestFlow(stdPath, tf)
		Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeNotFound),
			"Field": Equal("identifier[1].artifactsFrom"),
		}))))
	})

	It("should fail when step with artifactsFrom name does not exist", func() {
		tf := tmv1beta1.TestFlow{
			&tmv1beta1.DAGStep{
				Name: "int-test",
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test2",
				DependsOn: []string{"int-test"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
				ArtifactsFrom: "int-test3",
			},
		}
		errList := validation.ValidateTestFlow(stdPath, tf)
		Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeNotFound),
			"Field": Equal("identifier[1].artifactsFrom"),
		}))))
	})

	It("should fail artifactsFrom step name represents a preceding step", func() {
		tf := tmv1beta1.TestFlow{
			&tmv1beta1.DAGStep{
				Name: "int-test",
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test2",
				DependsOn: []string{"int-test"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
				ArtifactsFrom: "int-test",
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test3",
				DependsOn: []string{"int-test2"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
				ArtifactsFrom: "int-test4",
			},
			&tmv1beta1.DAGStep{
				Name:      "int-test4",
				DependsOn: []string{"int-test2"},
				Definition: tmv1beta1.StepDefinition{
					Name: "testdefname",
				},
			},
		}
		errList := validation.ValidateTestFlow(stdPath, tf)
		Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"Type":  Equal(field.ErrorTypeForbidden),
			"Field": Equal("identifier[2].artifactsFrom"),
		}))))
	})
})
