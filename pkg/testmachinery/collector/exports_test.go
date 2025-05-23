// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/util/json"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	mock_elasticsearch "github.com/gardener/test-infra/pkg/util/elasticsearch/mocks"
	mock_collector "github.com/gardener/test-infra/pkg/util/s3/mocks"
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
		tmpDir, err = os.MkdirTemp("", "test")
		Expect(err).ToNot(HaveOccurred())
		esCtrl = gomock.NewController(GinkgoT())
		s3Ctrl = gomock.NewController(GinkgoT())

		esClient = mock_elasticsearch.NewMockClient(esCtrl)
		s3Client = mock_collector.NewMockClient(s3Ctrl)
		c = &collector{
			log:      logr.Discard(),
			esClient: esClient,
			s3Client: s3Client,
			s3Config: &config.S3{BucketName: "testbucket"},
		}
	})

	AfterEach(func() {
		esCtrl.Finish()
		s3Ctrl.Finish()

		Expect(os.RemoveAll(tmpDir)).ToNot(HaveOccurred())
	})

	It("should add exported artifacts to the elasticsearch bulk output", func() {
		ctx := context.Background()
		defer ctx.Done()

		tr, err := testmachinery.ParseTestrunFromFile(filepath.Join(testdataDir, "02_testrun_export.yaml"))
		Expect(err).ToNot(HaveOccurred())

		s3Object, err := mock_collector.CreateS3ObjectFromFile(filepath.Join(testdataDir, "11_export_artifact.tar.gz"))
		Expect(err).ToNot(HaveOccurred())
		s3Client.EXPECT().GetObject("testbucket", "/testing/my/export.tar.gz").Return(s3Object, nil)

		err = c.collectSummaryAndExports(tmpDir, tr, &metadata.Metadata{Testrun: metadata.TestrunMetadata{ID: tr.Name}})
		Expect(err).ToNot(HaveOccurred())

		files, err := os.ReadDir(tmpDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(files)).To(Equal(1), "Expected 1 file output")

		file, err := os.Open(filepath.Join(tmpDir, files[0].Name()))
		Expect(err).ToNot(HaveOccurred())
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				fmt.Printf("Error closing file: %v", err)
			}
		}(file)

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
		Expect(lastDocument["name"]).To(Equal("test-export"))
	})

})
