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
			Expect(errList).To(HaveLen(0))
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
			Expect(errList).To(HaveLen(0))
		})
	})
})
