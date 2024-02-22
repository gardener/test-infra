// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testrun_renderer

import "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"

// TestsFunc generates tests that adds the parents as dependencies and return the generated steps
// and the names of the last steps that should be used as dependencies for subsequent steps.
type TestsFunc = func(suffix string, parents []string) ([]*v1beta1.DAGStep, []string, error)
