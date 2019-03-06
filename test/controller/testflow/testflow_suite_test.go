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
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	argoclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	tmclientset "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	maxWaitTime int64 = 300
)

func TestValidationWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testrun testflow Integration Test Suite")
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
		tmKubeconfig := os.Getenv("TM_KUBECONFIG_PATH")
		namespace = os.Getenv("TM_NAMESPACE")

		tmConfig, err := clientcmd.BuildConfigFromFlags("", tmKubeconfig)
		Expect(err).ToNot(HaveOccurred(), "couldn't create k8s client from kubeconfig filepath %s", tmKubeconfig)

		tmClient = tmclientset.NewForConfigOrDie(tmConfig)

		argoClient = argoclientset.NewForConfigOrDie(tmConfig)

		clusterClient, err = kubernetes.NewClientFromFile(tmKubeconfig, nil, client.Options{})
		Expect(err).ToNot(HaveOccurred())
		utils.WaitForClusterReadiness(clusterClient, namespace, maxWaitTime)
	})

	Context("testflow", func() {
		It("should run a test with TestDefs defined by name", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)

			tr, wf, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") && node.Phase == argov1.NodeSucceeded {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(1), "Testrun: %s", tr.Name)
		})

		It("should run a test with TestDefs defined by label and name", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = append(tr.Spec.TestFlow, []tmv1beta1.TestflowStep{
				{
					Label: "tm-integration",
				},
			})

			tr, wf, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") && node.Phase == argov1.NodeSucceeded {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(2), "Testrun: %s", tr.Name)
		})

		It("should execute all tests in right order when no testdefs for a label can be found", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = append(tr.Spec.TestFlow, []tmv1beta1.TestflowStep{
				{
					Name: "integration-testdef",
				},
				{
					Label: "tm-no-testdefs",
				},
				{
					Name: "integration-testdef",
				},
			})

			tr, wf, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") && node.Phase == argov1.NodeSucceeded {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(3), "Testrun: %s", tr.Name)
		})

		It("should execute serial steps after parallel steps", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
				{
					{
						Label: "tm-integration",
					},
				},
			}

			tr, wf, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())

			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") {
					Expect(node.Phase).To(Equal(argov1.NodeSucceeded))

					Expect(len(node.Children)).To(Equal(1))

					nextNode := wf.Status.Nodes[node.Children[0]]
					Expect(strings.HasPrefix(nextNode.TemplateName, "serial-testdef"))
				}
			}

			Expect(len(tr.Status.Steps[1])).To(Equal(1))
			Expect(tr.Status.Steps[1][0].TestDefinition.Name).To(Equal("serial-testdef"))
		})
	})

	Context("config", func() {
		Context("type environment variable", func() {
			It("should mount a config as environment variable", func() {
				tr := resources.GetBasicTestrun(namespace, commitSha)
				tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
					{
						{
							Name: "check-envvar-testdef",
						},
					},
				}
				tr.Spec.TestFlow[0][0].Config = []tmv1beta1.ConfigElement{
					{
						Type:  tmv1beta1.ConfigTypeEnv,
						Name:  "TEST_NAME",
						Value: "test",
					},
				}

				tr, _, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
				defer utils.DeleteTestrun(tmClient, tr)
				Expect(err).ToNot(HaveOccurred())

			})

			It("should mount a global config as environement variable", func() {
				tr := resources.GetBasicTestrun(namespace, commitSha)
				tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
					{
						{
							Name: "check-envvar-testdef",
						},
					},
				}
				tr.Spec.Config = []tmv1beta1.ConfigElement{
					{
						Type:  tmv1beta1.ConfigTypeEnv,
						Name:  "TEST_NAME",
						Value: "test",
					},
				}

				tr, _, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
				defer utils.DeleteTestrun(tmClient, tr)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("type file", func() {
			It("should mount a config with value as file to a specific path", func() {
				tr := resources.GetBasicTestrun(namespace, commitSha)
				tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
					{
						{
							Name: "check-file-testdef",
						},
					},
				}
				tr.Spec.TestFlow[0][0].Config = []tmv1beta1.ConfigElement{
					{
						Type:  tmv1beta1.ConfigTypeEnv,
						Name:  "FILE",
						Value: "/tmp/test",
					},
					{
						Type:  tmv1beta1.ConfigTypeFile,
						Name:  "TEST_NAME",
						Value: "dGVzdAo=", // base64 encoded 'test' string
						Path:  "/tmp/test",
					},
				}

				tr, _, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
				defer utils.DeleteTestrun(tmClient, tr)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should mount a config from a secret as file to a specific path", func() {
				ctx := context.Background()
				defer ctx.Done()
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "test-secret-",
						Namespace:    namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}
				err := clusterClient.Client().Create(ctx, secret)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err := clusterClient.Client().Delete(ctx, secret)
					Expect(err).ToNot(HaveOccurred(), "Cannot delete secret")
				}()

				tr := resources.GetBasicTestrun(namespace, commitSha)
				tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
					{
						{
							Name: "check-file-testdef",
						},
					},
				}
				tr.Spec.TestFlow[0][0].Config = []tmv1beta1.ConfigElement{
					{
						Type:  tmv1beta1.ConfigTypeEnv,
						Name:  "FILE",
						Value: "/tmp/test/test.txt",
					},
					{
						Type: tmv1beta1.ConfigTypeFile,
						Name: "TEST_NAME",
						Path: "/tmp/test/test.txt",
						ValueFrom: &tmv1beta1.ConfigSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secret.Name,
								},
								Key: "test",
							},
						},
					},
				}

				tr, _, err = utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
				defer utils.DeleteTestrun(tmClient, tr)
				Expect(err).ToNot(HaveOccurred())

			})

			It("should mount a config from a configmap as file to a specific path", func() {
				ctx := context.Background()
				defer ctx.Done()
				configmap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "test-configmap-",
						Namespace:    namespace,
					},
					Data: map[string]string{
						"test": "test",
					},
				}
				err := clusterClient.Client().Create(ctx, configmap)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err := clusterClient.Client().Delete(ctx, configmap)
					Expect(err).ToNot(HaveOccurred(), "Cannot delete configmap %s", configmap.Name)
				}()

				tr := resources.GetBasicTestrun(namespace, commitSha)
				tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
					{
						{
							Name: "check-file-testdef",
						},
					},
				}
				tr.Spec.TestFlow[0][0].Config = []tmv1beta1.ConfigElement{
					{
						Type:  tmv1beta1.ConfigTypeEnv,
						Name:  "FILE",
						Value: "/tmp/test/test.txt",
					},
					{
						Type: tmv1beta1.ConfigTypeFile,
						Name: "TEST_NAME",
						Path: "/tmp/test/test.txt",
						ValueFrom: &tmv1beta1.ConfigSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: configmap.Name,
								},
								Key: "test",
							},
						},
					},
				}

				tr, _, err = utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
				defer utils.DeleteTestrun(tmClient, tr)
				Expect(err).ToNot(HaveOccurred())

			})
		})

	})

	Context("onExit", func() {
		It("should run ExitHandlerTestDef when testflow succeeds", func() {
			tr := resources.GetTestrunWithExitHandler(resources.GetBasicTestrun(namespace, commitSha), tmv1beta1.ConditionTypeSuccess)

			tr, wf, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
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
			tr := resources.GetTestrunWithExitHandler(resources.GetBasicTestrun(namespace, commitSha), tmv1beta1.ConditionTypeError)

			tr, wf, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
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
			tr := resources.GetTestrunWithExitHandler(resources.GetFailingTestrun(namespace, commitSha), tmv1beta1.ConditionTypeError)

			tr, wf, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeFailed, namespace, maxWaitTime)
			defer utils.DeleteTestrun(tmClient, tr)
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

	Context("TTL", func() {
		It("should delete the testrun after ttl has finished", func() {
			var ttl int32 = 90
			var maxWaitTime int64 = 600
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TTLSecondsAfterFinished = &ttl

			tr, err := tmClient.Testmachinery().Testruns(tr.Namespace).Create(tr)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())

			startTime := time.Now()
			for !util.MaxTimeExceeded(startTime, maxWaitTime) {
				_, err := tmClient.Testmachinery().Testruns(tr.Namespace).Get(tr.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return
				}
				time.Sleep(5 * time.Second)
			}

			_, err = tmClient.Testmachinery().Testruns(tr.Namespace).Get(tr.Name, metav1.GetOptions{})
			Expect(errors.IsNotFound(err)).To(BeTrue(), "Testrun %s was not deleted in %d seconds", tr.Name, maxWaitTime)
		})

		It("should delete the testrun after workflow has finished", func() {
			var ttl int32 = 1
			var maxWaitTime int64 = 600
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TTLSecondsAfterFinished = &ttl

			tr, err := tmClient.Testmachinery().Testruns(tr.Namespace).Create(tr)
			defer utils.DeleteTestrun(tmClient, tr)
			Expect(err).ToNot(HaveOccurred())

			startTime := time.Now()
			for !util.MaxTimeExceeded(startTime, maxWaitTime) {
				_, err := tmClient.Testmachinery().Testruns(tr.Namespace).Get(tr.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return
				}
				time.Sleep(5 * time.Second)
			}
			_, err = tmClient.Testmachinery().Testruns(tr.Namespace).Get(tr.Name, metav1.GetOptions{})
			Expect(errors.IsNotFound(err)).To(BeTrue(), "Testrun %s was not deleted in %d seconds", tr.Name, maxWaitTime)
		})
	})
})
