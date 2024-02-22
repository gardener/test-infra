// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testrun

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1/validation"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/testmachinery/testflow"
)

// Validate validates a testrun.
// Returns if the validation can be retried
func Validate(log logr.Logger, tr *tmv1beta1.Testrun) (error, bool) {
	if allErrs := validation.ValidateTestrunSpec(tr.Spec); len(allErrs) != 0 {
		return allErrs.ToAggregate(), false
	}

	locs, err := locations.NewLocations(log, tr.Spec)
	if err != nil {
		return err, true
	}

	allErrs, retry := testflow.Validate(field.NewPath("spec", "testflow"), tr.Spec.TestFlow, locs, false)
	if errs, re := testflow.Validate(field.NewPath("spec", "onExit"), tr.Spec.TestFlow, locs, false); len(errs) != 0 {
		allErrs = append(allErrs, errs...)
		// only retry of both errors are retryable
		retry = retry && re
	}

	return allErrs.ToAggregate(), retry
}
