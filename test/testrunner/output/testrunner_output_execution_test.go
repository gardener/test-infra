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

package testrunner_output_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/gardener/test-infra/pkg/testrunner"

	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testrunner/result"

	"github.com/gardener/test-infra/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
		tmClient      kubernetes.Interface
		outputDirPath = "./out-"
		testrunConfig result.Config
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
		utils.WaitForMinioService(tmClient, s3Endpoint, namespace, maxWaitTime)
	})

	BeforeEach(func() {
		testrunConfig = result.Config{
			OutputDir:           ".",
			ESConfigName:        "es-config-name",
			S3Endpoint:          s3Endpoint,
			ConcourseOnErrorDir: ".",
		}
	})

	It("should output a summary of the testrun as elasticsearch bulk request", func() {
		ctx := context.Background()
		defer ctx.Done()
		testrunConfig.OutputDir = outputDirPath + util.RandomString(3)
		tr := resources.GetBasicTestrun(namespace, commitSha)
		tr, _, err := utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
		defer utils.DeleteTestrun(tmClient, tr)
		Expect(err).ToNot(HaveOccurred())

		err = result.Output(&testrunConfig, tmClient, namespace, tr, &testrunner.Metadata{Testrun: testrunner.TestrunMetadata{ID: tr.Name}})
		Expect(err).ToNot(HaveOccurred())

		files, err := ioutil.ReadDir(testrunConfig.OutputDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(files)).To(Equal(1), "Expected 1 file output")

		file, err := os.Open(filepath.Join(testrunConfig.OutputDir, files[0].Name()))
		Expect(err).ToNot(HaveOccurred())
		defer file.Close()
		defer func() {
			err := os.RemoveAll(testrunConfig.OutputDir)
			Expect(err).ToNot(HaveOccurred())
		}()

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
				Expect(jsonBody["tm"]).ToNot(BeEmpty())
				Expect(jsonBody["tm"].(map[string]interface{})["tr"]).ToNot(BeEmpty())

			}
			line++
		}
		Expect(scanner.Err()).ToNot(HaveOccurred())
	})

	It("should add exported artifacts to the elasticsearch bulk output", func() {
		ctx := context.Background()
		defer ctx.Done()
		testrunConfig.OutputDir = outputDirPath + util.RandomString(3)
		tr := resources.GetBasicTestrun(namespace, commitSha)
		tr, _, err := utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
		defer utils.DeleteTestrun(tmClient, tr)
		Expect(err).ToNot(HaveOccurred())

		err = result.Output(&testrunConfig, tmClient, namespace, tr, &testrunner.Metadata{Testrun: testrunner.TestrunMetadata{ID: tr.Name}})
		Expect(err).ToNot(HaveOccurred())

		files, err := ioutil.ReadDir(testrunConfig.OutputDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(files)).To(Equal(1), "Expected 1 file output")

		file, err := os.Open(filepath.Join(testrunConfig.OutputDir, files[0].Name()))
		Expect(err).ToNot(HaveOccurred())
		defer file.Close()
		defer func() {
			err := os.RemoveAll(testrunConfig.OutputDir)
			Expect(err).ToNot(HaveOccurred())
		}()

		documents := []map[string]interface{}{}
		scanner := bufio.NewScanner(file)
		var jsonBody map[string]interface{}
		for scanner.Scan() {
			err = json.Unmarshal([]byte(scanner.Text()), &jsonBody)
			Expect(err).ToNot(HaveOccurred())

			documents = append(documents, jsonBody)
		}
		Expect(scanner.Err()).ToNot(HaveOccurred())

		Expect(jsonBody["tm"]).ToNot(BeNil())
		Expect(jsonBody["tm"].(map[string]interface{})["tr"].(map[string]interface{})["id"]).To(Equal(tr.Name))

		Expect(documents[len(documents)-2]["index"].(map[string]interface{})["_index"]).To(Equal("integration-testdef"))
	})

	It("should add environemnt configuration to step metadata", func() {
		ctx := context.Background()
		defer ctx.Done()

		configElement := tmv1beta1.ConfigElement{
			Type:  tmv1beta1.ConfigTypeEnv,
			Name:  "test",
			Value: "val",
		}

		testrunConfig.OutputDir = outputDirPath + util.RandomString(3)
		tr := resources.GetBasicTestrun(namespace, commitSha)
		tr.Spec.TestFlow[0].Definition.Config = []tmv1beta1.ConfigElement{configElement}
		tr, _, err := utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
		defer utils.DeleteTestrun(tmClient, tr)
		Expect(err).ToNot(HaveOccurred())

		err = result.Output(&testrunConfig, tmClient, namespace, tr, &testrunner.Metadata{Testrun: testrunner.TestrunMetadata{ID: tr.Name}})
		Expect(err).ToNot(HaveOccurred())

		files, err := ioutil.ReadDir(testrunConfig.OutputDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(files)).To(Equal(1), "Expected 1 file output")

		file, err := os.Open(filepath.Join(testrunConfig.OutputDir, files[0].Name()))
		Expect(err).ToNot(HaveOccurred())
		defer file.Close()
		defer func() {
			err := os.RemoveAll(testrunConfig.OutputDir)
			Expect(err).ToNot(HaveOccurred())
		}()

		documents := []map[string]interface{}{}
		scanner := bufio.NewScanner(file)
		var jsonBody map[string]interface{}
		for scanner.Scan() {
			err = json.Unmarshal([]byte(scanner.Text()), &jsonBody)
			Expect(err).ToNot(HaveOccurred())

			documents = append(documents, jsonBody)
		}
		Expect(scanner.Err()).ToNot(HaveOccurred())

		Expect(jsonBody["tm"]).ToNot(BeNil())
		Expect(jsonBody["tm"].(map[string]interface{})["tr"].(map[string]interface{})["id"]).To(Equal(tr.Name))

		Expect(documents[len(documents)-2]["index"].(map[string]interface{})["_index"]).To(Equal("integration-testdef"))
	})

})
