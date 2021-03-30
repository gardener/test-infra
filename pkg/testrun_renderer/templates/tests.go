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

package templates

import (
	"fmt"
	"strings"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrun_renderer"
	"github.com/gardener/test-infra/pkg/util"
)

// TestWithLabels creates tests functions that render test steps executed in serial.
func TestWithName(name string) testrun_renderer.TestsFunc {
	return func(suffix string, parents []string) ([]*v1beta1.DAGStep, []string, error) {
		step := GetTestStepWithName(fmt.Sprintf("tests-%s-%s", suffix, util.RandomString(3)), name, parents)
		return []*v1beta1.DAGStep{&step}, []string{step.Name}, nil
	}
}

// TestWithLabels creates tests functions that render test steps executed in serial.
func TestWithLabels(labels ...string) testrun_renderer.TestsFunc {
	return func(suffix string, parents []string) ([]*v1beta1.DAGStep, []string, error) {
		steps := make([]*v1beta1.DAGStep, len(labels))
		previous := parents
		for i, l := range labels {
			step := GetTestStepWithLabels(fmt.Sprintf("tests-%s-%s", suffix, util.RandomString(3)), previous, l)
			steps[i] = &step
			previous = []string{step.Name}
		}

		return steps, []string{steps[len(steps)-1].Name}, nil
	}
}

// HibernationLifecycle returns a testcase that tests
// - the hibernation of a shoot
// - waking up of a shoot
// - rehibernation of a shoot
func HibernationLifecycle(suffix string, parents []string) ([]*v1beta1.DAGStep, []string, error) {
	hibernate := GetTestStepWithName(fmt.Sprintf("hibernate-%s", suffix), "hibernate-shoot", parents)
	wakeup := GetTestStepWithName(fmt.Sprintf("wakeup-%s", suffix), "wakeup-shoot", []string{hibernate.Name})
	hibernateAgain := GetTestStepWithName(fmt.Sprintf("hibernate-again-%s", suffix), "hibernate-shoot", []string{wakeup.Name})
	return []*v1beta1.DAGStep{&hibernate, &wakeup, &hibernateAgain}, []string{hibernateAgain.Name}, nil
}

// GetTestStepWithName returns a default test step for a specific testdefinition
func GetTestStepWithName(name, testName string, dependencies []string) v1beta1.DAGStep {
	if name == "" {
		name = "tests"
	}
	return v1beta1.DAGStep{
		Name: name,
		Definition: v1beta1.StepDefinition{
			Name: testName,
		},
		UseGlobalArtifacts: false,
		DependsOn:          dependencies,
	}
}

// GetTestStepWithLabels returns a default test step for all testdefinitions with a specific label
func GetTestStepWithLabels(name string, dependencies []string, labels ...string) v1beta1.DAGStep {
	if name == "" {
		name = "tests"
	}
	return v1beta1.DAGStep{
		Name: name,
		Definition: v1beta1.StepDefinition{
			Label: strings.Join(labels, ","),
		},
		UseGlobalArtifacts: false,
		DependsOn:          dependencies,
	}
}
