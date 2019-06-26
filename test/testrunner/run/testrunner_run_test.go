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
	"time"

	"github.com/gardener/test-infra/pkg/testmachinery"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testrunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var (
	maxWaitTime = 300 * time.Second
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
		tmClient      kubernetes.Interface
		testrunConfig testrunner.Config
		s3Endpoint    string
	)

	BeforeSuite(func() {
		var err error
		commitSha = os.Getenv("GIT_COMMIT_SHA")
		tmKubeconfig = os.Getenv("TM_KUBECONFIG_PATH")
		namespace = os.Getenv("TM_NAMESPACE")
		s3Endpoint = os.Getenv("S3_ENDPOINT")

		tmClient, err = kubernetes.NewClientFromFile("", tmKubeconfig, client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(utils.WaitForClusterReadiness(tmClient, namespace, maxWaitTime)).ToNot(HaveOccurred())
		_, err = utils.WaitForMinioService(tmClient, s3Endpoint, namespace, maxWaitTime)
		Expect(err).ToNot(HaveOccurred())
	})

	BeforeEach(func() {
		testrunConfig = testrunner.Config{
			TmClient:  tmClient,
			Namespace: namespace,
			Timeout:   int64(maxWaitTime),
			Interval:  5,
		}
	})

	Context("testrun", func() {
		It("should run a single testrun", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			run := []*testrunner.Run{
				{
					Testrun:  tr,
					Metadata: &testrunner.Metadata{},
				},
			}
			finishedTr, err := testrunner.ExecuteTestrun(&testrunConfig, run, "test-")
			defer utils.DeleteTestrun(tmClient, finishedTr[0].Testrun)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(finishedTr)).To(Equal(1))
			Expect(finishedTr[0].Testrun.Status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))
		})

		It("should run 2 testruns", func() {
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr2 := resources.GetBasicTestrun(namespace, commitSha)
			run := []*testrunner.Run{
				{
					Testrun:  tr,
					Metadata: &testrunner.Metadata{},
				},
				{
					Testrun:  tr2,
					Metadata: &testrunner.Metadata{},
				},
			}
			finishedTr, err := testrunner.ExecuteTestrun(&testrunConfig, run, "test-")
			defer utils.DeleteTestrun(tmClient, finishedTr[0].Testrun)
			defer utils.DeleteTestrun(tmClient, finishedTr[1].Testrun)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(finishedTr)).To(Equal(2))
			Expect(finishedTr[0].Testrun.Status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))
			Expect(finishedTr[1].Testrun.Status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))
		})

	})

})
