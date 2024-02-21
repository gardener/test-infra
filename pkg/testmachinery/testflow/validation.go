// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testflow

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1/validation"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
)

// Validate validates a testrun and all its subcomponenets.
// This function validate in addition to the default validation function also the testlocations.
// Returns true if the operation cvan be retried.
// todo: refactor this to use better errors
func Validate(fldPath *field.Path, tf tmv1beta1.TestFlow, locs locations.Locations, ignoreEmptyFlow bool) (field.ErrorList, bool) {
	var (
		usedTestdefinitions = 0
		usedStepNames       = make(map[string]*tmv1beta1.DAGStep)
		allErrs             field.ErrorList
		retry               bool
	)

	for i, step := range tf {
		stepPath := fldPath.Index(i)

		testDefinitions, err := locs.GetTestDefinitions(step.Definition)
		if err != nil {
			allErrs = append(allErrs, field.InternalError(stepPath.Child("definition"), err))
			retry = true
			continue
		}

		// fail if there are no testdefinitions found
		if len(testDefinitions) == 0 && !ignoreEmptyFlow {
			allErrs = append(allErrs, field.Required(stepPath.Child("definition"), "no TestDefinitions found for step"))
			retry = true
			continue
		}

		for _, td := range testDefinitions {
			tdPath := stepPath.Child(fmt.Sprintf("Location: %q; File: %q", td.Location.Name(), td.FileName))
			allErrs = append(allErrs, validation.ValidateTestDefinition(tdPath, td.Info)...)
		}

		usedStepNames[step.Name] = step
		usedTestdefinitions += len(testDefinitions)
	}

	// check if there are any testruns to execute. Fail if there are none.
	if !ignoreEmptyFlow && usedTestdefinitions == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, nil, "no testdefinitions found"))
		retry = true
	}

	return allErrs, retry
}
