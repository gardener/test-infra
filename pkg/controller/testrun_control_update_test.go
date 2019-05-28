package controller

import (
	"testing"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testmachinery Controller Suite")
}

var (
	workflowTmpl argov1.Workflow
	testrunTmpl  tmv1beta1.Testrun
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
						TemplateName: "template1",
						Phase:        argov1.NodeRunning,
					},
				},
			},
		}
	})

	Context("Update status", func() {
		It("should update the status of 1 step and 1 template", func() {
			tr := testrunTmpl
			wf := workflowTmpl
			updateStepsStatus(&tr, &wf)
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
					TemplateName: "template1",
					Phase:        argov1.NodeSucceeded,
				},
				"node2": {
					TemplateName: "template2",
					Phase:        argov1.NodeFailed,
				},
				"node3": {
					TemplateName: "template4",
					Phase:        argov1.NodeSucceeded,
				},
				"node4": {
					TemplateName: "template3",
					Phase:        argov1.NodeRunning,
				},
			}
			updateStepsStatus(&tr, &wf)
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
