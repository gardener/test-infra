// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1/validation"
	"github.com/gardener/test-infra/pkg/util/strconf"
)

var _ = Describe("Config", func() {

	Context("Validating config elements", func() {

		It("should fail without a config name", func() {
			elem := tmv1beta1.ConfigElement{
				Type: "en",
			}
			errList := validation.ValidateConfig(stdPath, elem)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("identifier.name"),
			}))))
		})

		It("should fail with no value or valueFrom defined", func() {
			elem := tmv1beta1.ConfigElement{
				Name: "testConfig",
				Type: "env",
			}
			errList := validation.ValidateConfig(stdPath, elem)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("identifier.value/valueFrom"),
			}))))
		})

		It("should fail when valueFrom is defined but no config or secret ref is provided", func() {
			elem := tmv1beta1.ConfigElement{
				Name:      "testConfig",
				Type:      "env",
				ValueFrom: &strconf.ConfigSource{},
			}
			errList := validation.ValidateConfig(stdPath, elem)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("identifier.valueFrom.configMapKeyRef/secretMapKeyRef"),
			}))))
		})

		It("should succeed when valueFrom is defined and a config ref is provided", func() {
			elem := tmv1beta1.ConfigElement{
				Name: "testConfig",
				Type: "env",
				ValueFrom: &strconf.ConfigSource{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						Key: "test",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "name",
						},
					},
				},
			}
			errList := validation.ValidateConfig(stdPath, elem)
			Expect(errList).To(BeEmpty())
		})

		It("should fail with unknown config type", func() {
			elem := tmv1beta1.ConfigElement{
				Name: "testConfig",
				Type: "en",
			}
			errList := validation.ValidateConfig(stdPath, elem)
			Expect(errList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("identifier.type"),
			}))))
		})

		It("should succeed with known config type", func() {
			elem := tmv1beta1.ConfigElement{
				Name:  "testConfig",
				Type:  "env",
				Value: "this is a value",
			}
			errList := validation.ValidateConfig(stdPath, elem)
			Expect(errList).To(BeEmpty())
		})
	})
})
