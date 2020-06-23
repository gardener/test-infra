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

package resultcollection_test

import (
	"context"
	"fmt"
	"time"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var _ = Describe("Result collection tests", func() {

	Context("step status", func() {
		It("should update status immediately with all steps of the generated testflow", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())

			err := operation.Client().Client().Create(ctx, tr)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			// todo: add polling instead of timeouts
			time.Sleep(10 * time.Second)
			err = operation.Client().Client().Get(ctx, client.ObjectKey{Namespace: operation.TestNamespace(), Name: tr.Name}, tr)
			Expect(err).ToNot(HaveOccurred())

			Expect(tr.Status.Steps).To(HaveLen(1), "Should be one steps status")
			status := tr.Status.Steps[0]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.Phase).To(Equal(tmv1beta1.PhaseStatusInit), "Tests should be initially in 'init' phase")
		})

		It("should collect status of all workflow nodes", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.NodeSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			Expect(tr.Status.Steps).To(HaveLen(1), "Should be one steps status")

			status := tr.Status.Steps[0]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.ExportArtifactKey).ToNot(BeZero())
			Expect(status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))

		})

		It("should collect configurations for steps", func() {
			ctx := context.Background()
			defer ctx.Done()

			configElement := tmv1beta1.ConfigElement{
				Type:  tmv1beta1.ConfigTypeEnv,
				Name:  "test",
				Value: "val",
			}

			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow[0].Definition.Config = []tmv1beta1.ConfigElement{configElement}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.NodeSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			Expect(tr.Status.Steps).To(HaveLen(1), "Should be one steps status")

			status := tr.Status.Steps[0]
			Expect(status.TestDefinition.Config).To(ContainElement(&configElement))
		})

		It("should collect status of multiple workflow nodes", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				{
					Name: "A",
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
				},
				{
					Name:      "C",
					DependsOn: []string{"A"},
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
				},
			}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.NodeSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			Expect(tr.Status.Steps).To(HaveLen(2), "Should be 2 step's statuses.")

			status := tr.Status.Steps[0]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.ExportArtifactKey).ToNot(BeZero())
			Expect(status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))

			status = tr.Status.Steps[1]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.ExportArtifactKey).ToNot(BeZero())
			Expect(status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))

		})

		It("should collect status of multiple workflow nodes with a failing step", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				{
					Name: "A",
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
				},
				{
					Name: "B",
					Definition: tmv1beta1.StepDefinition{
						Name: "failing-integration-testdef",
					},
				},
				{
					Name:      "C",
					DependsOn: []string{"A", "B"},
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
				},
			}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.NodeFailed, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			Expect(tr.Status.Steps).To(HaveLen(3), "Should be 3 step's statuses.")

			for _, status := range tr.Status.Steps {
				Expect(status.TestDefinition.Name).To(Or(Equal("integration-testdef"), Equal("failing-integration-testdef")))

				if status.TestDefinition.Name == "failing-integration-testdef" {
					Expect(status.ExportArtifactKey).To(BeZero()) // needs to be zero as argo does not upload empty tars anymore (> v2.3.0)
					Expect(status.Phase).To(Equal(tmv1beta1.PhaseStatusFailed))
				}

				if status.Position.Step == "A" {
					Expect(status.ExportArtifactKey).ToNot(BeZero(), fmt.Sprintf("string is %s not zero valued", status.ExportArtifactKey))
				}

				if status.Position.Step == "C" {
					Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
					Expect(status.ExportArtifactKey).To(BeZero())
					Expect(status.Phase).To(Equal(tmv1beta1.PhaseStatusSkipped))
				}
			}
		})

		It("should mark timeouted step with own timeout phase", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "timeout-integration-testdef",
					},
				},
			}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.NodeFailed, TestrunDurationTimeout)
			//defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			Expect(tr.Status.Steps).To(HaveLen(1), "Should be one steps status")

			status := tr.Status.Steps[0]
			Expect(status.TestDefinition.Name).To(Equal("timeout-integration-testdef"))
			Expect(status.Phase).To(Equal(tmv1beta1.PhaseStatusTimeout))

		})
	})
})
