// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package collector

import (
	"bufio"
	"context"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	mock_elasticsearch "github.com/gardener/test-infra/pkg/util/elasticsearch/mocks"
	mock_collector "github.com/gardener/test-infra/pkg/util/s3/mocks"
	"github.com/golang/mock/gomock"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testdataDir = "./testdata"
)

var _ = Describe("collector summary", func() {

	var (
		tmpDir   string
		esCtrl   *gomock.Controller
		s3Ctrl   *gomock.Controller
		esClient *mock_elasticsearch.MockClient
		s3Client *mock_collector.MockClient
		c        *collector
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "test")
		Expect(err).ToNot(HaveOccurred())
		esCtrl = gomock.NewController(GinkgoT())
		s3Ctrl = gomock.NewController(GinkgoT())

		esClient = mock_elasticsearch.NewMockClient(esCtrl)
		s3Client = mock_collector.NewMockClient(s3Ctrl)
		c = &collector{
			log:      log.NullLogger{},
			esClient: esClient,
			s3Client: s3Client,
			s3Config: &testmachinery.S3Config{BucketName: "testbucket"},
		}
	})

	AfterEach(func() {
		esCtrl.Finish()
		s3Ctrl.Finish()

		Expect(os.RemoveAll(tmpDir)).ToNot(HaveOccurred())
	})

	It("should output a summary of the testrun as elasticsearch bulk request", func() {
		ctx := context.Background()
		defer ctx.Done()

		tr, err := testmachinery.ParseTestrunFromFile(filepath.Join(testdataDir, "01_testrun.yaml"))
		Expect(err).ToNot(HaveOccurred())

		err = c.collectSummaryAndExports(tmpDir, tr, &metadata.Metadata{Testrun: metadata.TestrunMetadata{ID: tr.Name}})
		Expect(err).ToNot(HaveOccurred())

		files, err := ioutil.ReadDir(tmpDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(files)).To(Equal(1), "Expected 1 file output")

		file, err := os.Open(filepath.Join(tmpDir, files[0].Name()))
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
				Expect(jsonBody["tm"]).ToNot(BeEmpty())
				Expect(jsonBody["tm"].(map[string]interface{})["tr"]).ToNot(BeEmpty())

			}
			line++
		}
		Expect(scanner.Err()).ToNot(HaveOccurred())
	})

	It("should add environment configuration to step metadata", func() {
		ctx := context.Background()
		defer ctx.Done()

		tr, err := testmachinery.ParseTestrunFromFile(filepath.Join(testdataDir, "01_testrun.yaml"))
		Expect(err).ToNot(HaveOccurred())

		err = c.collectSummaryAndExports(tmpDir, tr, &metadata.Metadata{Testrun: metadata.TestrunMetadata{ID: tr.Name}})
		Expect(err).ToNot(HaveOccurred())

		files, err := ioutil.ReadDir(tmpDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(files)).To(Equal(1), "Expected 1 file output")

		file, err := os.Open(filepath.Join(tmpDir, files[0].Name()))
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

		Expect(jsonBody["tm"]).ToNot(BeNil())
		Expect(jsonBody["tm"].(map[string]interface{})["tr"].(map[string]interface{})["id"]).To(Equal(tr.Name))

		lastDocument := documents[len(documents)-1]
		Expect(lastDocument["tm"].(map[string]interface{})["config"].(map[string]interface{})["test"]).To(Equal("val"))
	})

	It("should upload a summary of the testrun and exports as elasticsearch bulk request", func() {
		ctx := context.Background()
		defer ctx.Done()

		tr, err := testmachinery.ParseTestrunFromFile(filepath.Join(testdataDir, "02_testrun_export.yaml"))
		Expect(err).ToNot(HaveOccurred())
		s3Object, err := mock_collector.CreateS3ObjectFromFile(filepath.Join(testdataDir, "11_export_artifact.tar.gz"))
		Expect(err).ToNot(HaveOccurred())
		s3Client.EXPECT().GetObject("testbucket", "/testing/my/export.tar.gz", gomock.Any()).Return(s3Object, nil)

		// esClient.Request(http.MethodGet, "/testmachinery-*/_search", strings.NewReader(payload))
		hits := `{ "hits": { "total": { "value": 0 } } }`
		esClient.EXPECT().Request(http.MethodGet, "/testmachinery-*/_search", gomock.Any()).Return([]byte(hits), nil)
		esClient.EXPECT().BulkFromFile(gomock.AssignableToTypeOf("")).Return(nil)

		err = c.Collect(tr, &metadata.Metadata{Testrun: metadata.TestrunMetadata{ID: tr.Name}})
		Expect(err).ToNot(HaveOccurred())

		Expect(tr.Status.Collected).To(BeTrue())

	})
})
