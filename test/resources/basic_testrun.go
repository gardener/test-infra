// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var basicTestrun = &tmv1beta1.Testrun{
	ObjectMeta: metav1.ObjectMeta{
		GenerateName: "integration-test-tm",
	},
	TypeMeta: metav1.TypeMeta{
		Kind:       "Testrun",
		APIVersion: "testmachinery.sapcloud.io/v1beta1",
	},
	Spec: tmv1beta1.TestrunSpec{
		Creator: "tm-integration",
		LocationSets: []tmv1beta1.LocationSet{
			{
				Name:    "default",
				Default: true,
				Locations: []tmv1beta1.TestLocation{
					{
						Type:     tmv1beta1.LocationTypeGit,
						Repo:     "https://github.com/gardener/test-infra.git",
						Revision: "master",
					},
				},
			},
		},
		TestFlow: tmv1beta1.TestFlow{
			&tmv1beta1.DAGStep{
				Name: "A",
				Definition: tmv1beta1.StepDefinition{
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
	tr.Spec.LocationSets[0].Locations[0].Revision = commitSha
	return tr
}

// GetFailingTestrun returns a testrun with a failing test.
func GetFailingTestrun(namespace, commitSha string) *tmv1beta1.Testrun {
	tr := GetBasicTestrun(namespace, commitSha)
	tr.Spec.TestFlow = tmv1beta1.TestFlow{
		&tmv1beta1.DAGStep{
			Name: "failing",
			Definition: tmv1beta1.StepDefinition{
				Name: "failing-integration-testdef",
			},
		},
	}
	return tr
}

// GetTestrunWithExitHandler returns a working testrun object with an onExit handler with a specific condition.
func GetTestrunWithExitHandler(tr *tmv1beta1.Testrun, condition tmv1beta1.ConditionType) *tmv1beta1.Testrun {
	tr.Spec.OnExit = tmv1beta1.TestFlow{
		&tmv1beta1.DAGStep{
			Name: "failing",
			Definition: tmv1beta1.StepDefinition{
				Name:      "exit-handler-testdef",
				Condition: condition,
			},
		},
	}
	return tr
}
