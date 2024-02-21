// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testflow_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/util/strconf"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var _ = Describe("Testflow execution tests", func() {

	Context("kubeconfigs string", func() {
		It("should add a shoot kubeconfig defined by a base64 encoded string to all test steps", func() {
			ctx := context.Background()
			defer ctx.Done()

			// get kubeconfigfile from testdata
			file, err := os.ReadFile("./testdata/kubeconfig")
			Expect(err).ToNot(HaveOccurred())

			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.Kubeconfigs.Shoot = strconf.FromString(base64.StdEncoding.EncodeToString(file))
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "check-file-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: filepath.Join(testmachinery.TM_KUBECONFIG_PATH, tmv1beta1.ShootKubeconfigName),
							},
						},
					},
				},
			}

			tr, _, err = operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})

	})

	Context("kubeconfigs config source", func() {
		It("should add a shoot kubeconfig referenced from a secret to all test steps", func() {
			ctx := context.Background()
			defer ctx.Done()

			// get kubeconfigfile from testdata
			file, err := os.ReadFile("./testdata/kubeconfig")
			Expect(err).ToNot(HaveOccurred())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-kubeconfig-",
					Namespace:    operation.TestNamespace(),
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"kubeconfig": file,
				},
			}
			err = operation.Client().Create(ctx, secret)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err := operation.Client().Delete(ctx, secret)
				Expect(err).ToNot(HaveOccurred(), "Cannot delete secret")
			}()
			operation.Log().Info(fmt.Sprintf("created secret %s", secret.Name))

			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.Kubeconfigs.Shoot = strconf.FromConfig(strconf.ConfigSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secret.Name,
					},
					Key: "kubeconfig",
				},
			})
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "check-file-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: filepath.Join(testmachinery.TM_KUBECONFIG_PATH, tmv1beta1.ShootKubeconfigName),
							},
						},
					},
				},
			}

			tr, _, err = operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("kubeconfigs", func() {
		It("should add a shoot kubeconfig to all test steps and add a gardener kubeconfig only to trusted", func() {
			ctx := context.Background()
			defer ctx.Done()

			// get kubeconfigfile from testdata
			file, err := os.ReadFile("./testdata/kubeconfig")
			Expect(err).ToNot(HaveOccurred())

			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.Kubeconfigs.Gardener = strconf.FromString(base64.StdEncoding.EncodeToString(file))
			tr.Spec.Kubeconfigs.Shoot = strconf.FromString(base64.StdEncoding.EncodeToString(file))
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "trusted-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "check-file-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: filepath.Join(testmachinery.TM_KUBECONFIG_PATH, tmv1beta1.GardenerKubeconfigName),
							},
						},
					},
				},
				&tmv1beta1.DAGStep{
					Name: "untrusted-test",
					Definition: tmv1beta1.StepDefinition{
						Name:      "check-file-not-exist-testdef",
						Untrusted: true,
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: filepath.Join(testmachinery.TM_KUBECONFIG_PATH, tmv1beta1.GardenerKubeconfigName),
							},
						},
					},
				},
			}

			tr, _, err = operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})

	})
})
