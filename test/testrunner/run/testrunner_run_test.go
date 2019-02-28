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

package testrunner_test

import (
	"os"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testrunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	tmclientset "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var (
	maxWaitTime int64 = 300
)

func TestValidationWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testrunner Integration Test Suite")
}

var _ = Describe("Testrunner execution tests", func() {

	var (
		commitSha     string
		namespace     string
		tmKubeconfig  string
		tmClient      *tmclientset.Clientset
		testrunConfig testrunner.Config
		s3Endpoint    string
	)

	BeforeSuite(func() {
		var err error
		commitSha = os.Getenv("GIT_COMMIT_SHA")
		tmKubeconfig = os.Getenv("TM_KUBECONFIG_PATH")
		namespace = os.Getenv("TM_NAMESPACE")
		s3Endpoint = os.Getenv("S3_ENDPOINT")

		tmConfig, err := clientcmd.BuildConfigFromFlags("", tmKubeconfig)
		Expect(err).ToNot(HaveOccurred(), "couldn't create k8s client from kubeconfig filepath %s", tmKubeconfig)

		tmClient = tmclientset.NewForConfigOrDie(tmConfig)

		clusterClient, err := kubernetes.NewClientFromFile(tmKubeconfig, nil, client.Options{})
		Expect(err).ToNot(HaveOccurred())

		utils.WaitForClusterReadiness(clusterClient, namespace, maxWaitTime)
		utils.WaitForMinioService(clusterClient, s3Endpoint, namespace, maxWaitTime)
	})

	BeforeEach(func() {
		testrunConfig = testrunner.Config{
			TmKubeconfigPath: tmKubeconfig,
			Namespace:        namespace,
			Timeout:          maxWaitTime,
			Interval:         5,
		}
	})

	Context("testrun", func() {
		It("should run a single testrun", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			finishedTr, err := testrunner.Run(&testrunConfig, []*tmv1beta1.Testrun{tr}, "test-")
			defer utils.DeleteTestrun(tmClient, finishedTr[0])
			Expect(err).ToNot(HaveOccurred())

			Expect(len(finishedTr)).To(Equal(1))
			Expect(finishedTr[0].Status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))
		})

		It("should run 2 testruns", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr2 := resources.GetBasicTestrun(namespace, commitSha)
			finishedTr, err := testrunner.Run(&testrunConfig, []*tmv1beta1.Testrun{tr, tr2}, "test-")
			defer utils.DeleteTestrun(tmClient, finishedTr[0])
			defer utils.DeleteTestrun(tmClient, finishedTr[1])
			Expect(err).ToNot(HaveOccurred())

			Expect(len(finishedTr)).To(Equal(2))
			Expect(finishedTr[0].Status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))
			Expect(finishedTr[1].Status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))
		})

	})

})
