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

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

// Validate validates a testrun and all its subcomponenets.
func Validate(identifier string, tf *tmv1beta1.TestFlow, tl testdefinition.TestDefinitions, ignoreEmptyFlow bool) error {

	usedTestdefinitions := 0

	for i, steps := range *tf {
		for j, item := range steps {
			step := item

			identifier := fmt.Sprintf("%s.[%d].[%d]", identifier, i, j)

			if step.Label == "" && step.Name == "" {
				return fmt.Errorf("%s: Required value: name or label have to be specified", identifier)
			}

			if step.Name != "" {
				if err := testdefinition.ValidateName(identifier, step.Name); err != nil {
					return err
				}
			}

			if step.Condition != tmv1beta1.ConditionTypeAlways &&
				step.Condition != tmv1beta1.ConditionTypeSuccess &&
				step.Condition != tmv1beta1.ConditionTypeError &&
				step.Condition != "" {
				return fmt.Errorf("%s.type: Unsupported condition type \"%s\"", identifier, step.Condition)
			}

			for elemIndex, elem := range step.Config {
				if err := config.Validate(fmt.Sprintf("%s.config.[%d]", identifier, elemIndex), elem); err != nil {
					return err
				}
			}

			testdefinitions, err := tl.GetTestDefinitions(&step)
			if err != nil {
				return err
			}

			for _, td := range testdefinitions {
				if err := testdefinition.Validate(fmt.Sprintf("%s; Location: \"%s\"; File: \"%s\"", identifier, td.Location.Name(), td.FileName), td.Info); err != nil {
					return err
				}
			}

			usedTestdefinitions += len(testdefinitions)
		}
	}

	// check if there are any testruns to execute. Fail if there are none.
	if !ignoreEmptyFlow && usedTestdefinitions == 0 {
		return fmt.Errorf("%s: No testdefinitions found", identifier)
	}

	return nil
}
