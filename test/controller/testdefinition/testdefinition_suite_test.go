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

package testdefinition_test

import (
	"context"
	"os"
	"testing"

	"github.com/gardener/test-infra/pkg/testmachinery"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var (
	maxWaitTime int64 = 300
)

func TestValidationWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testrun testdefinition Integration Test Suite")
}

var _ = Describe("Testrun tests", func() {

	var (
		commitSha string
		namespace string
		tmClient  kubernetes.Interface
	)

	BeforeSuite(func() {
		var err error
		commitSha = os.Getenv("GIT_COMMIT_SHA")
		namespace = os.Getenv("TM_NAMESPACE")
		tmKubeconfig := os.Getenv("TM_KUBECONFIG_PATH")

		tmClient, err = kubernetes.NewClientFromFile("", tmKubeconfig, client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(utils.WaitForClusterReadiness(tmClient, namespace, maxWaitTime)).ToNot(HaveOccurred())
	})

	Context("config", func() {
		Context("type environment variable", func() {
			It("should run a TesDef with a environment variable defined by value", func() {
				ctx := context.Background()
				defer ctx.Done()
				tr := resources.GetBasicTestrun(namespace, commitSha)
				tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
					{
						{
							Name: "value-config-testdef",
						},
					},
				}

				tr, _, err := utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
				defer utils.DeleteTestrun(tmClient, tr)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should run a TesDef with a environment variable defined by a secret", func() {
				ctx := context.Background()
				defer ctx.Done()
				tr := resources.GetBasicTestrun(namespace, commitSha)
				tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
					{
						{
							Name: "secret-config-testdef",
						},
					},
				}

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}
				err := tmClient.Client().Create(ctx, secret)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err := tmClient.Client().Delete(ctx, secret)
					Expect(err).ToNot(HaveOccurred(), "Cannot delete secret")
				}()

				tr, _, err = utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
				defer utils.DeleteTestrun(tmClient, tr)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("type file", func() {
			It("should run a TesDef with a file defined by a secret", func() {
				ctx := context.Background()
				defer ctx.Done()
				tr := resources.GetBasicTestrun(namespace, commitSha)
				tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
					{
						{
							Name: "secret-config-file-testdef",
						},
					},
				}

				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret-file",
						Namespace: namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}
				err := tmClient.Client().Create(ctx, secret)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err := tmClient.Client().Delete(ctx, secret)
					Expect(err).ToNot(HaveOccurred(), "Cannot delete secret")
				}()

				tr, _, err = utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
				defer utils.DeleteTestrun(tmClient, tr)
				Expect(err).ToNot(HaveOccurred())
			})
		})

	})
})
