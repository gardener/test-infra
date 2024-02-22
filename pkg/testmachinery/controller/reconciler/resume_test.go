// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"time"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

var _ = Describe("Testmachinery controller resume", func() {

	BeforeEach(func() {
		reconciler = &TestmachineryReconciler{
			Logger: logr.Discard(),
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
				&tmv1beta1.DAGStep{
					Name:       "step1",
					Definition: tmv1beta1.StepDefinition{Name: "testdef1"},
				},
				&tmv1beta1.DAGStep{
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
					Phase: tmv1beta1.StepPhaseInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef1",
					},
				},
				{
					Name: "template2",
					Position: tmv1beta1.StepStatusPosition{
						Step: "step2",
					},
					Phase: tmv1beta1.StepPhaseInit,
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
