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

package testflow

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/locations"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

func ValidateDefinition(identifier string, definition tmv1beta1.StepDefinition) error {
	if definition.Label == "" && definition.Name == "" {
		return fmt.Errorf("%s: Required value: name or label have to be specified", identifier)
	}
	if definition.Name != "" {
		if err := testdefinition.ValidateName(identifier, definition.Name); err != nil {
			return err
		}
	}

	if definition.Condition != tmv1beta1.ConditionTypeAlways &&
		definition.Condition != tmv1beta1.ConditionTypeSuccess &&
		definition.Condition != tmv1beta1.ConditionTypeError &&
		definition.Condition != "" {
		return fmt.Errorf("%s.type: Unsupported condition type \"%s\"", identifier, definition.Condition)
	}

	for elemIndex, elem := range definition.Config {
		if err := config.Validate(fmt.Sprintf("%s.config.[%d]", identifier, elemIndex), elem); err != nil {
			return err
		}
	}

	return nil
}

// Validate validates a testrun and all its subcomponenets.
func Validate(identifier string, tf tmv1beta1.TestFlow, locs locations.Locations, ignoreEmptyFlow bool) error {

	usedTestdefinitions := 0

	usedStepNames := make(map[string]bool, 0)

	for i, step := range tf {
		identifier := fmt.Sprintf("%s.[%d]", identifier, i)

		if step.Name == "" {
			return fmt.Errorf("%s.Name: Required value: Name has to be defined", identifier)
		}

		if _, ok := usedStepNames[step.Name]; ok {
			return fmt.Errorf("%s.Name: Name %s needs to be unique", identifier, step.Name)
		}

		if err := ValidateDefinition(identifier, step.Definition); err != nil {
			return err
		}

		testdefinitions, err := locs.GetTestDefinitions(step.Definition)
		if err != nil {
			return err
		}

		for _, td := range testdefinitions {
			if err := testdefinition.Validate(fmt.Sprintf("%s; Location: \"%s\"; File: \"%s\"", identifier, td.Location.Name(), td.FileName), td.Info); err != nil {
				return err
			}
		}

		usedStepNames[step.Name] = true
		usedTestdefinitions += len(testdefinitions)
	}

	// check if dependent steps exist
	for i, step := range tf {
		identifier := fmt.Sprintf("%s.[%d]", identifier, i)
		if step.DependsOn != nil && len(step.DependsOn) != 0 {
			for _, name := range step.DependsOn {
				if _, ok := usedStepNames[name]; !ok {
					return fmt.Errorf("%s.dependsOn: Invalid value: Step %s is unknown", identifier, name)
				}
			}
		}
	}

	// check if there are any testruns to execute. Fail if there are none.
	if !ignoreEmptyFlow && usedTestdefinitions == 0 {
		return fmt.Errorf("%s: No testdefinitions found", identifier)
	}

	return nil
}
