// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testflow_test

import (
	"context"
	"time"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var _ = Describe("Testflow execution tests", func() {

	Context("resume", func() {
		It("should resume a step within the specified timeout", func() {
			var (
				err           error
				resumeTimeout = 20
			)
			ctx := context.Background()
			defer ctx.Done()

			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = append(tr.Spec.TestFlow, &tmv1beta1.DAGStep{
				Name:      "B",
				DependsOn: []string{"A"},
				Definition: tmv1beta1.StepDefinition{
					Name: "integration-testdef",
				},
				Pause: &tmv1beta1.Pause{
					Enabled:              true,
					ResumeTimeoutSeconds: &resumeTimeout,
				},
			})

			tr, _, err = operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not resume before the timeout has finished", func() {
			var (
				err error
			)
			ctx := context.Background()
			defer ctx.Done()

			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "A",
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
					Pause: &tmv1beta1.Pause{
						Enabled: true,
					},
				},
			}

			tr, _, err = operation.RunTestrun(ctx, tr, tmv1beta1.RunPhaseRunning, 2*time.Minute, utils.WatchUntil(2*time.Minute))
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})

	})
})
