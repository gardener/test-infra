// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"testing"
	"time"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testmachinery Controller Suite")
}

var (
	workflowTmpl argov1.Workflow
	testrunTmpl  tmv1beta1.Testrun
	reconciler   *TestmachineryReconciler
)

var _ = BeforeSuite(func() {
	testrunTmpl = tmv1beta1.Testrun{
		Status: tmv1beta1.TestrunStatus{
			Steps: []*tmv1beta1.StepStatus{
				{
					Name:  "template1",
					Phase: tmv1beta1.StepPhaseInit,
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

var _ = Describe("Testmachinery controller update", func() {

	BeforeEach(func() {
		reconciler = &TestmachineryReconciler{
			Logger: logr.Discard(),
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
					Phase: tmv1beta1.StepPhaseInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef1",
					},
				},
				{
					Name:  "template2",
					Phase: tmv1beta1.StepPhaseInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef2",
					},
				},
				{
					Name:  "template3",
					Phase: tmv1beta1.StepPhaseInit,
					TestDefinition: tmv1beta1.StepStatusTestDefinition{
						Name: "testdef3",
					},
				},
				{
					Name:  "template4",
					Phase: tmv1beta1.StepPhaseInit,
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
