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

package validationwebhook_test

import (
	"os"

	"github.com/gardener/test-infra/test/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	tmclientset "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/test-infra/test/resources"
	"k8s.io/client-go/tools/clientcmd"
)

var _ = Describe("Testrun validation tests", func() {

	var (
		commitSha   string
		namespace   string
		tmClient    *tmclientset.Clientset
		maxWaitTime int64 = 300
	)

	BeforeSuite(func() {
		var err error
		commitSha = os.Getenv("GIT_COMMIT_SHA")
		tmKubeconfig := os.Getenv("TM_KUBECONFIG_PATH")
		namespace = os.Getenv("TM_NAMESPACE")

		tmConfig, err := clientcmd.BuildConfigFromFlags("", tmKubeconfig)
		Expect(err).ToNot(HaveOccurred(), "couldn't create k8s client from kubeconfig filepath %s", tmKubeconfig)

		tmClient = tmclientset.NewForConfigOrDie(tmConfig)

		clusterClient, err := kubernetes.NewClientFromFile(tmKubeconfig, nil, client.Options{})
		Expect(err).ToNot(HaveOccurred())
		utils.WaitForClusterReadiness(clusterClient, namespace, maxWaitTime)

	})

	Context("Metadata", func() {
		It("should reject when name contains '.'", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
				[]tmv1beta1.TestflowStep{
					tmv1beta1.TestflowStep{
						Name: "integration.testdef",
					},
				},
			}
			_, err := tmClient.Testmachinery().Testruns(namespace).Create(tr)
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("name must not contain '.'"))
		})
	})

	Context("TestLocations", func() {
		It("should reject when no locations are defined", func() {

			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestLocations = []tmv1beta1.TestLocation{}

			_, err := tmClient.Testmachinery().Testruns(namespace).Create(tr)
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("No TestDefinition locations defined"))
		})

		It("should reject when a local location is defined", func() {

			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestLocations = append(tr.Spec.TestLocations, tmv1beta1.TestLocation{
				Type: tmv1beta1.LocationTypeLocal,
			})

			tr, err := tmClient.Testmachinery().Testruns(namespace).Create(tr)
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("Local testDefinition locations are only available in insecure mode"))
		})
	})

	Context("Testflow", func() {
		It("should reject when no locations can be found", func() {

			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{}

			tr, err := tmClient.Testmachinery().Testruns(namespace).Create(tr)
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("No testdefinitions found"))
		})

		It("should reject when a no locations for only one label in the testrun can be found", func() {

			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = [][]tmv1beta1.TestflowStep{
				[]tmv1beta1.TestflowStep{
					tmv1beta1.TestflowStep{
						Label: "NoTestDefsFoundLabel",
					},
				},
			}

			tr, err := tmClient.Testmachinery().Testruns(namespace).Create(tr)
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("No testdefinitions found"))
		})
	})

	Context("Kubeconfigs", func() {
		It("should reject when a invalid kubeconfig is provided", func() {

			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.Kubeconfigs.Gardener = "dGVzdGluZwo="

			tr, err := tmClient.Testmachinery().Testruns(namespace).Create(tr)
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}
			Expect(err).To(HaveOccurred())
			Expect(string(errors.ReasonForError(err))).To(ContainSubstring("Cannot build config"))
		})
	})

	Context("OnExit", func() {
		It("should accept when no steps are defined", func() {

			tr := resources.GetBasicTestrun(namespace, commitSha)

			tr, err := tmClient.Testmachinery().Testruns(namespace).Create(tr)
			defer utils.DeleteTestrun(tmClient, tr)

			Expect(err).ToNot(HaveOccurred())

			_, err = tmClient.Testmachinery().Testruns(namespace).Get(tr.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

	})

})
