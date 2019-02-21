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

package resources

import (
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var basicTestrun = &tmv1beta1.Testrun{
	ObjectMeta: metav1.ObjectMeta{
		GenerateName: "integration-test-tm",
	},
	Spec: tmv1beta1.TestrunSpec{
		Creator: "tm-integration",
		TestLocations: []tmv1beta1.TestLocation{
			{
				Type:     tmv1beta1.LocationTypeGit,
				Repo:     "https://github.com/gardener/test-infra.git",
				Revision: "master",
			},
		},
		TestFlow: [][]tmv1beta1.TestflowStep{
			{
				{
					Name: "integration-testdef",
				},
			},
		},
	},
}

// GetBasicTestrun returns a working testrun object with a specific namespace
func GetBasicTestrun(namespace, commitSha string) *tmv1beta1.Testrun {
	tr := basicTestrun.DeepCopy()
	tr.Namespace = namespace
	tr.Spec.TestLocations[0].Revision = commitSha
	return tr
}

// GetFailingTestrun returns a testrun with a failing test.
func GetFailingTestrun(namespace, commitSha string) *tmv1beta1.Testrun {
	tr := GetBasicTestrun(namespace, commitSha)
	tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
		{
			{
				Name: "failing-integration-testdef",
			},
		},
	}
	return tr
}

// GetTestrunWithExitHandler returns a working testrun object with an onExit handler with a specific condition.
func GetTestrunWithExitHandler(tr *tmv1beta1.Testrun, condition tmv1beta1.ConditionType) *tmv1beta1.Testrun {
	tr.Spec.OnExit = [][]tmv1beta1.TestflowStep{
		{
			{
				Name:      "exit-handler-testdef",
				Condition: condition,
			},
		},
	}
	return tr
}
