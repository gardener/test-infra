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
	"github.com/gardener/test-infra/pkg/testmachinery"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

var _ = Describe("Testmachinery controller resume", func() {

	BeforeEach(func() {
		reconciler = &TestrunReconciler{
			Logger: log.NullLogger{},
			timers: make(map[string]*time.Timer),
		}
	})

	Context("Resume", func() {
		It("should calculate the timer duration", func() {
			var (
				now      = metav1.Now()
				timeout  = 10 * time.Minute
				elapsed  = -5 * time.Minute
				expected = 5 * time.Minute
			)

			actual := calculateTimer(timeout, metav1.NewTime(now.Add(elapsed)))
			Expect(actual.Seconds()).To(BeNumerically("~", expected.Seconds(), 1e-3))
		})

		It("should add a timer if a step is paused", func() {
			tr := testrunTmpl
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				{
					Name:       "step1",
					Definition: tmv1beta1.StepDefinition{Name: "testdef1"},
				},
				{
					Name:       "step2",
					Definition: tmv1beta1.StepDefinition{Name: "testdef2"},
					Pause: &tmv1beta1.Pause{
						Enabled:              true,
						ResumeTimeoutSeconds: nil,
					},
				},
			}
			tr.Status.Steps = []*tmv1beta1.StepStatus{
				{
					Name: "template1",
					Position: tmv1beta1.StepStatusPosition{
						Step: "step1",
					},
					Phase: tmv1beta1.PhaseStatusInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef1",
					},
				},
				{
					Name: "template2",
					Position: tmv1beta1.StepStatusPosition{
						Step: "step2",
					},
					Phase: tmv1beta1.PhaseStatusInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef2",
					},
				},
			}

			wf := workflowTmpl
			wf.Status.Nodes = map[string]argov1.NodeStatus{
				"node1": {
					DisplayName:  "template1",
					TemplateName: "template1",
					Phase:        argov1.NodeSucceeded,
				},
				"node2": {
					DisplayName:  testmachinery.GetPauseTaskName("template2"),
					TemplateName: testmachinery.PauseTemplateName,
					Phase:        argov1.NodeRunning,
					StartedAt:    metav1.Now(),
				},
				"node3": {
					DisplayName:  "template2",
					TemplateName: "template2",
				},
			}
			reconciler.updateStepsStatus(&reconcileContext{
				tr:      &tr,
				wf:      &wf,
				updated: false,
			})

			Expect(reconciler.timers).To(HaveLen(1))

		})
	})
})
