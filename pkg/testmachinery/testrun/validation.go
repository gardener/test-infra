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
func Validate(log logr.Logger, tr *tmv1beta1.Testrun) error {
	var allErrs field.ErrorList
	// validate locations
	allErrs = append(allErrs, validation.ValidateTestrunSpec(tr.Spec)...)

	if len(allErrs) != 0 {
		return allErrs.ToAggregate()
	}

	locs, err := locations.NewLocations(log, tr.Spec)
	if err != nil {
		return err
	}

	allErrs = append(allErrs, testflow.Validate(field.NewPath("spec", "testflow"), tr.Spec.TestFlow, locs, false)...)
	allErrs = append(allErrs, testflow.Validate(field.NewPath("spec", "onExit"), tr.Spec.TestFlow, locs, false)...)

	return allErrs.ToAggregate()
}
