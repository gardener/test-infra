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
	"bufio"
	"encoding/json"
	"os"

	"github.com/gardener/test-infra/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/cmd/testmachinery-run/testrunner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	argoclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	tmclientset "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"

	"k8s.io/client-go/tools/clientcmd"
)

var (
	maxWaitTime int64 = 300
)

var _ = Describe("Testrunner execution tests", func() {

	var (
		commitSha      string
		namespace      string
		tmKubeconfig   string
		tmClient       *tmclientset.Clientset
		argoClient     *argoclientset.Clientset
		outputFilePath = "./out-"
		testrunConfig  testrunner.TestrunConfig
	)

	BeforeSuite(func() {
		var err error
		commitSha = os.Getenv("GIT_COMMIT_SHA")
		tmKubeconfig = os.Getenv("TM_KUBECONFIG_PATH")
		namespace = os.Getenv("TM_NAMESPACE")

		tmConfig, err := clientcmd.BuildConfigFromFlags("", tmKubeconfig)
		Expect(err).ToNot(HaveOccurred(), "couldn't create k8s client from kubeconfig filepath %s", tmKubeconfig)

		tmClient = tmclientset.NewForConfigOrDie(tmConfig)

		argoClient = argoclientset.NewForConfigOrDie(tmConfig)

		clusterClient, err := kubernetes.NewClientFromFile(tmKubeconfig, nil, client.Options{})
		Expect(err).ToNot(HaveOccurred())

		utils.WaitForClusterReadiness(clusterClient, namespace, maxWaitTime)

	})

	BeforeEach(func() {
		var timeout int64 = -1
		testrunConfig = testrunner.TestrunConfig{
			TmKubeconfigPath:     tmKubeconfig,
			GardenKubeconfigPath: "",
			Timeout:              &timeout,
			OutputFile:           ".",
			ESConfigName:         "es-config-name",
			S3Endpoint:           "S3_ENDPOINT",
			ConcourseOnErrorDir:  ".",
		}
	})

	Context("output", func() {
		It("should output a summary of the testrun as elasticsearch bulk request", func() {
			testrunConfig.OutputFile = outputFilePath + util.RandomString(3)
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr, _, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			Expect(err).ToNot(HaveOccurred())
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}

			err = testrunner.Output(&testrunConfig, tr, &testrunner.Metadata{})
			Expect(err).ToNot(HaveOccurred())

			file, err := os.Open(testrunConfig.OutputFile)
			Expect(err).ToNot(HaveOccurred())
			defer file.Close()

			scanner := bufio.NewScanner(file)
			line := 1
			for scanner.Scan() {
				var jsonBody map[string]interface{}
				err = json.Unmarshal([]byte(scanner.Text()), &jsonBody)
				Expect(err).ToNot(HaveOccurred())

				// every second json should be a elastic search metadat file
				if line%2 != 0 {
					Expect(jsonBody["index"]).ToNot(BeNil())
				}
				// every data document should have of a testrun metadata information
				if line%2 == 0 {
					Expect(jsonBody["index"]).To(BeNil())
					Expect(jsonBody["testrun_id"] != nil || jsonBody["tm_meta"] != nil).To(BeTrue())
				}
				line++
			}
			Expect(scanner.Err()).ToNot(HaveOccurred())
		})

		It("should add exported artifacts to the elasticsearch bulk ouput", func() {
			testrunConfig.OutputFile = outputFilePath + util.RandomString(3)
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr, _, err := utils.RunTestrun(tmClient, argoClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
			Expect(err).ToNot(HaveOccurred())
			if err == nil {
				defer utils.DeleteTestrun(tmClient, tr)
			}

			err = testrunner.Output(&testrunConfig, tr, &testrunner.Metadata{})
			Expect(err).ToNot(HaveOccurred())

			file, err := os.Open(testrunConfig.OutputFile)
			Expect(err).ToNot(HaveOccurred())
			defer file.Close()

			documents := []map[string]interface{}{}
			scanner := bufio.NewScanner(file)
			var jsonBody map[string]interface{}
			for scanner.Scan() {
				err = json.Unmarshal([]byte(scanner.Text()), &jsonBody)
				Expect(err).ToNot(HaveOccurred())

				documents = append(documents, jsonBody)
			}
			Expect(scanner.Err()).ToNot(HaveOccurred())

			Expect(jsonBody["tm_meta"]).ToNot(BeNil())
			Expect(jsonBody["tm_meta"].(map[string]interface{})["testrun_id"]).To(Equal(tr.Name))

			Expect(documents[len(documents)-2]["index"].(map[string]interface{})["_index"]).To(Equal("integration-testdef"))
		})

	})

})
