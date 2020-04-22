// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package testrun_renderer

import "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"

// WithTests connects mumtiple test functions to one
func WithTests(funcs ...TestsFunc) TestsFunc {
	return func(suffix string, parents []string) ([]*v1beta1.DAGStep, []string, error) {
		steps := make([]*v1beta1.DAGStep, 0)
		dependencies := parents
		for _, f := range funcs {
			var (
				testSteps []*v1beta1.DAGStep
				err       error
			)
			testSteps, dependencies, err = f(suffix, dependencies)
			if err != nil {
				return nil, nil, err
			}
			steps = append(steps, testSteps...)
		}

		return steps, dependencies, nil
	}
}
