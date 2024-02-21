// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testflow_test

import (
	"context"
	"strings"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var _ = Describe("testflow exit tests", func() {

	Context("onExit", func() {
		It("should not run ExitHandlerTestDef when testflow succeeds", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetTestrunWithExitHandler(resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit()), tmv1beta1.ConditionTypeSuccess)

			tr, wf, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "exit-handler-testdef") && node.Phase == argov1.NodeSucceeded {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(1), "Testrun: %s", tr.Name)
		})

		It("should not run exit-handler-testdef when testflow succeeds", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetTestrunWithExitHandler(resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit()), tmv1beta1.ConditionTypeError)

			tr, wf, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "exit-handler-testdef") && node.Phase != argov1.NodeSkipped {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(0), "Testrun: %s", tr.Name)
		})

		It("should run exit-handler-testdef when testflow fails", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetTestrunWithExitHandler(resources.GetFailingTestrun(operation.TestNamespace(), operation.Commit()), tmv1beta1.ConditionTypeError)

			tr, wf, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowFailed, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "exit-handler-testdef") && node.Phase != argov1.NodeSkipped {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(1), "Testrun: %s", tr.Name)
		})
	})

	Context("phase", func() {
		It("testmachinery phase should be propagated to default and onExit testflow with its right values", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "check-dynamic-envvar-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "ENV_NAME",
								Value: testmachinery.TM_PHASE_NAME,
							},
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "ENV_VALUE",
								Value: string(testmachinery.PhaseRunning),
							},
						},
					},
				},
			}
			tr.Spec.OnExit = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "check-dynamic-envvar-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "ENV_NAME",
								Value: testmachinery.TM_PHASE_NAME,
							},
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "ENV_VALUE",
								Value: string(testmachinery.PhaseExit),
							},
						},
					},
				},
			}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

		})
	})

})
