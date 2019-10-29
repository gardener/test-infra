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

package controller

import (
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testmachinery Controller Suite")
}

var (
	workflowTmpl argov1.Workflow
	testrunTmpl  tmv1beta1.Testrun
	reconciler   *TestrunReconciler
)

var _ = Describe("Testmachinery controller update", func() {

	BeforeSuite(func() {
		testrunTmpl = tmv1beta1.Testrun{
			Status: tmv1beta1.TestrunStatus{
				Steps: []*tmv1beta1.StepStatus{
					{
						Name:  "template1",
						Phase: tmv1beta1.PhaseStatusInit,
						TestDefinition: tmv1beta1.StepStatusTestDefinition{
							Name: "testdef1",
						},
					},
				},
			},
		}
		workflowTmpl = argov1.Workflow{
			Spec: argov1.WorkflowSpec{
				Templates: []argov1.Template{
					{
						Name: "template1",
					},
				},
			},
			Status: argov1.WorkflowStatus{
				Nodes: map[string]argov1.NodeStatus{
					"node1": {
						DisplayName: "template1",
						Phase:       argov1.NodeRunning,
					},
				},
			},
		}
	})

	BeforeEach(func() {
		reconciler = &TestrunReconciler{
			Logger: log.NullLogger{},
			timers: make(map[string]*time.Timer),
		}
	})

	Context("Update status", func() {
		It("should update the status of 1 step and 1 template", func() {
			tr := testrunTmpl
			wf := workflowTmpl
			reconciler.updateStepsStatus(&reconcileContext{
				tr:      &tr,
				wf:      &wf,
				updated: false,
			})
			Expect(tr.Status.Steps).To(Equal([]*tmv1beta1.StepStatus{
				{
					Name:  "template1",
					Phase: argov1.NodeRunning,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef1",
					},
				},
			}))
		})

		It("should update the status of multiple steps and templates", func() {
			tr := testrunTmpl
			tr.Status.Steps = []*tmv1beta1.StepStatus{
				{
					Name:  "template1",
					Phase: tmv1beta1.PhaseStatusInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef1",
					},
				},
				{
					Name:  "template2",
					Phase: tmv1beta1.PhaseStatusInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef2",
					},
				},
				{
					Name:  "template3",
					Phase: tmv1beta1.PhaseStatusInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef3",
					},
				},
				{
					Name:  "template4",
					Phase: tmv1beta1.PhaseStatusInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef2",
					},
				},
			}
			wf := workflowTmpl

			wf.Status.Nodes = map[string]argov1.NodeStatus{
				"node1": {
					DisplayName: "template1",
					Phase:       argov1.NodeSucceeded,
				},
				"node2": {
					DisplayName: "template2",
					Phase:       argov1.NodeFailed,
				},
				"node3": {
					DisplayName: "template4",
					Phase:       argov1.NodeSucceeded,
				},
				"node4": {
					DisplayName: "template3",
					Phase:       argov1.NodeRunning,
				},
			}
			reconciler.updateStepsStatus(&reconcileContext{
				tr:      &tr,
				wf:      &wf,
				updated: false,
			})
			Expect(tr.Status.Steps).To(Equal([]*tmv1beta1.StepStatus{
				{
					Name:  "template1",
					Phase: argov1.NodeSucceeded,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef1",
					},
				},
				{
					Name:  "template2",
					Phase: argov1.NodeFailed,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef2",
					},
				},
				{
					Name:  "template3",
					Phase: argov1.NodeRunning,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef3",
					},
				},
				{
					Name:  "template4",
					Phase: argov1.NodeSucceeded,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef2",
					},
				},
			}))
		})
	})
})
