// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// ValidateTestrun validates a testrun.
func ValidateTestrun(tr *tmv1beta1.Testrun) error {
	errList := ValidateTestrunSpec(tr.Spec)

	if len(errList) == 0 {
		return nil
	}

	return errors.NewInvalid(
		schema.GroupKind{
			Group: tmv1beta1.SchemeGroupVersion.Group,
			Kind:  tr.GetObjectKind().GroupVersionKind().Kind},
		tr.Name,
		errList,
	)
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
