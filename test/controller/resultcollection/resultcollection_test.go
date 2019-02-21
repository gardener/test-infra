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
	"os"
	"time"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argoclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	tmclientset "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	maxWaitTime int64 = 300
)

var _ = Describe("Result collection tests", func() {

	var (
		commitSha  string
		namespace  string
		tmClient   *tmclientset.Clientset
		argoClient *argoclientset.Clientset
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

		clusterClient, err := kubernetes.NewClientFromFile(tmKubeconfig, nil, client.Options{})
		Expect(err).ToNot(HaveOccurred())

		utils.WaitForClusterReadiness(clusterClient, namespace, maxWaitTime)

	})

	Context("step status", func() {
		It("should update status immediately with all steps of the generated testflow", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)

			tr, err := tmClient.Testmachinery().Testruns(tr.Namespace).Create(tr)
			Expect(err).ToNot(HaveOccurred())
			defer utils.DeleteTestrun(tmClient, tr)

			time.Sleep(5 * time.Second)
			tr, err = tmClient.Testmachinery().Testruns(namespace).Get(tr.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(utils.TestflowLen(tr.Status.Steps)).To(Equal(1), "Should be one steps status")
			status := tr.Status.Steps[0][0]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.Phase).To(Equal(tmv1beta1.PhaseStatusInit), "Tests should be initially in 'init' phase")
		})

		It("should collect status of all workflow nodes", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)

			tr, _, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			Expect(err).ToNot(HaveOccurred())
			defer utils.DeleteTestrun(tmClient, tr)

			Expect(utils.TestflowLen(tr.Status.Steps)).To(Equal(1), "Should be one steps status")

			status := tr.Status.Steps[0][0]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.ExportArtifactKey).ToNot(BeZero())
			Expect(status.Phase).To(Equal(argov1.NodeSucceeded))

		})

		It("should collect status of multiple workflow nodes", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
				{
					{
						Name: "integration-testdef",
					},
					{
						Label: "tm-no-testdefs",
					},
				},
				{
					{
						Name: "integration-testdef",
					},
				},
			}

			tr, _, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			Expect(err).ToNot(HaveOccurred())
			defer utils.DeleteTestrun(tmClient, tr)

			Expect(utils.TestflowLen(tr.Status.Steps)).To(Equal(2), "Should be 2 step's statuses.")

			status := tr.Status.Steps[0][0]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.ExportArtifactKey).ToNot(BeZero())
			Expect(status.Phase).To(Equal(argov1.NodeSucceeded))

			status = tr.Status.Steps[1][0]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.ExportArtifactKey).ToNot(BeZero())
			Expect(status.Phase).To(Equal(argov1.NodeSucceeded))

		})

		It("should collect status of multiple workflow nodes with a failing step", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
				{
					{
						Name: "integration-testdef",
					},
					{
						Name: "failing-integration-testdef",
					},
				},
				{
					{
						Name: "integration-testdef",
					},
				},
			}

			tr, _, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeFailed, namespace, maxWaitTime)
			Expect(err).ToNot(HaveOccurred())
			defer utils.DeleteTestrun(tmClient, tr)

			Expect(utils.TestflowLen(tr.Status.Steps)).To(Equal(3), "Should be 3 step's statuses.")

			status := tr.Status.Steps[0][0]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.ExportArtifactKey).ToNot(BeZero())

			status = tr.Status.Steps[0][1]
			Expect(status.TestDefinition.Name).To(Equal("failing-integration-testdef"))
			Expect(status.ExportArtifactKey).ToNot(BeZero())
			Expect(status.Phase).To(Equal(argov1.NodeFailed))

			status = tr.Status.Steps[1][0]
			Expect(status.TestDefinition.Name).To(Equal("integration-testdef"))
			Expect(status.ExportArtifactKey).To(BeZero())
			Expect(status.Phase).To(Equal(argov1.NodeSkipped))
		})
	})
})
