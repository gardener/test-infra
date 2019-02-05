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
	"os"
	"testing"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	tmclientset "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	maxWaitTime int64 = 300
)

func TestValidationWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testrun testdefinition Integration Test Suite")
}

var _ = Describe("Testflow execution tests", func() {

	var (
		commitSha     string
		namespace     string
		tmClient      *tmclientset.Clientset
		argoClient    *argoclientset.Clientset
		clusterClient kubernetes.Interface
	)

	BeforeSuite(func() {
		var err error
		commitSha = os.Getenv("GIT_COMMIT_SHA")
		namespace = os.Getenv("TM_NAMESPACE")
		tmKubeconfig := os.Getenv("TM_KUBECONFIG_PATH")

		tmConfig, err := clientcmd.BuildConfigFromFlags("", tmKubeconfig)
		Expect(err).ToNot(HaveOccurred(), "couldn't create k8s client from kubeconfig filepath %s", tmKubeconfig)

		tmClient = tmclientset.NewForConfigOrDie(tmConfig)

		argoClient = argoclientset.NewForConfigOrDie(tmConfig)

		clusterClient, err = kubernetes.NewClientFromFile(tmKubeconfig, nil, client.Options{})
		Expect(err).ToNot(HaveOccurred())
		utils.WaitForClusterReadiness(clusterClient, namespace, maxWaitTime)

	})

	Context("config", func() {
		It("should run a TesDef with a evironment variable defined by value", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
				[]tmv1beta1.TestflowStep{
					tmv1beta1.TestflowStep{
						Name: "ValueConfigTestDef",
					},
				},
			}

			tr, _, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			Expect(err).ToNot(HaveOccurred())
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}
		})

		It("should run a TesDef with a evironment variable defined by a secret", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
				[]tmv1beta1.TestflowStep{
					tmv1beta1.TestflowStep{
						Name: "SecretConfigTestDef",
					},
				},
			}

			_, err := clusterClient.CreateSecret(namespace, "test-secret", corev1.SecretTypeOpaque, map[string][]byte{
				"test": []byte("test"),
			}, false)
			Expect(err).ToNot(HaveOccurred())
			defer func() {
				err := clusterClient.DeleteSecret(namespace, "test-secret")
				Expect(err).ToNot(HaveOccurred(), "Cannot delete secrett")
			}()

			tr, _, err = utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			Expect(err).ToNot(HaveOccurred())
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}
		})

	})
})
