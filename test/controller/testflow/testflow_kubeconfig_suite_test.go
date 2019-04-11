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
	"encoding/base64"
	"io/ioutil"
	"path/filepath"

	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/util/strconf"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Testflow execution tests", func() {

	Context("kubeconfigs string", func() {
		It("should add a shoot kubeconfig defined by a base64 encoded string to all test steps", func() {
			ctx := context.Background()
			defer ctx.Done()

			// get kubeconfigfile from testdata
			file, err := ioutil.ReadFile("./testdata/kubeconfig")
			Expect(err).ToNot(HaveOccurred())

			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.Kubeconfigs.Shoot = strconf.FromString(base64.StdEncoding.EncodeToString(file))
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
				{
					{
						Name: "check-file-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: filepath.Join(testmachinery.TM_KUBECONFIG_PATH, "shoot.config"),
							},
						},
					},
				},
			}

			tr, _, err = utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())
		})

	})

	Context("kubeconfigs configsrouce", func() {
		It("should add a shoot kubeconfig referenced from a secret to all test steps", func() {
			ctx := context.Background()
			defer ctx.Done()

			// get kubeconfigfile from testdata
			file, err := ioutil.ReadFile("./testdata/kubeconfig")
			Expect(err).ToNot(HaveOccurred())

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-kubeconfig",
					Namespace:    namespace,
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"kubeconfig": file,
				},
			}
			err = tmClient.Client().Create(ctx, secret)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err := tmClient.Client().Delete(ctx, secret)
				Expect(err).ToNot(HaveOccurred(), "Cannot delete secret")
			}()

			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.Kubeconfigs.Shoot = strconf.FromConfig(strconf.ConfigSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secret.Name,
					},
					Key: "kubeconfig",
				},
			})
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
				{
					{
						Name: "check-file-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: filepath.Join(testmachinery.TM_KUBECONFIG_PATH, "shoot.config"),
							},
						},
					},
				},
			}

			tr, _, err = utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())
		})

	})
})
