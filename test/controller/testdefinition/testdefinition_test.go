// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testdefinition_test

import (
	"context"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var _ = Describe("Testrun tests", func() {

	Context("config", func() {
		Context("type environment variable", func() {
			It("should run a TesDef with a environment variable defined by value", func() {
				ctx := context.Background()
				defer ctx.Done()
				tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
				tr.Spec.TestFlow = tmv1beta1.TestFlow{
					&tmv1beta1.DAGStep{
						Name: "int-test",
						Definition: tmv1beta1.StepDefinition{
							Name: "value-config-testdef",
						},
					},
				}

				tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
				defer utils.DeleteTestrun(operation.Client(), tr)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should run a TesDef with a environment variable defined by a secret", func() {
				ctx := context.Background()
				defer ctx.Done()
				tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
				tr.Spec.TestFlow = tmv1beta1.TestFlow{
					&tmv1beta1.DAGStep{
						Name: "int-test",
						Definition: tmv1beta1.StepDefinition{
							Name: "secret-config-testdef",
						},
					},
				}

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: operation.TestNamespace(),
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}
				err := operation.Client().Create(ctx, secret)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err := operation.Client().Delete(ctx, secret)
					Expect(err).ToNot(HaveOccurred(), "Cannot delete secret")
				}()

				tr, _, err = operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
				defer utils.DeleteTestrun(operation.Client(), tr)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("type file", func() {
			It("should run a TesDef with a file defined by a secret", func() {
				ctx := context.Background()
				defer ctx.Done()
				tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
				tr.Spec.TestFlow = tmv1beta1.TestFlow{
					&tmv1beta1.DAGStep{
						Name: "int-test",
						Definition: tmv1beta1.StepDefinition{
							Name: "secret-config-file-testdef",
						},
					},
				}

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret-file",
						Namespace: operation.TestNamespace(),
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}
				err := operation.Client().Create(ctx, secret)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err := operation.Client().Delete(ctx, secret)
					Expect(err).ToNot(HaveOccurred(), "Cannot delete secret")
				}()

				tr, _, err = operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
				defer utils.DeleteTestrun(operation.Client(), tr)
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})
})
