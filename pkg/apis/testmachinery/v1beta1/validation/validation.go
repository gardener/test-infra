// Copyright 2021 Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// ValidateTestrun validates a testrun.
func ValidateTestrun(tr *tmv1beta1.Testrun) error {
	errList := ValidateTestrunSpec(tr.Spec)
	return errList.ToAggregate()
}

// ValidateTestrunSpec validates a testrun spec
func ValidateTestrunSpec(spec tmv1beta1.TestrunSpec) field.ErrorList {
	var (
		fldPath = field.NewPath("spec")
		allErrs field.ErrorList
	)

	allErrs = append(allErrs, ValidateConfigList(fldPath.Child("configs"), spec.Config)...)
	allErrs = append(allErrs, ValidateLocations(fldPath, spec)...)
	allErrs = append(allErrs, ValidateKubeconfigs(fldPath.Child("kubeconfigs"), spec.Kubeconfigs)...)

	allErrs = append(allErrs, ValidateTestFlow(fldPath.Child("testflow"), spec.TestFlow)...)
	allErrs = append(allErrs, ValidateTestFlow(fldPath.Child("onExit"), spec.OnExit)...)

	return allErrs
}
