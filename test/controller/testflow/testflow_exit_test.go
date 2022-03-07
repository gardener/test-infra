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

package testflow_test

import (
	"context"
	"strings"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/testmachinery"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
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
