// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package bulk

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestES(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ElasticsearchExport Suite")
}

var (
	tmMeta = map[string]interface{}{
		"tm_id": "testId",
	}
)

var _ = Describe("elasticsearch test", func() {

	Context("Parse default json format", func() {
		It("should parse default json and add testmachinery index", func() {
			input, err := os.ReadFile("./testdata/json")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/json")

			output, err := os.ReadFile("./testdata/json_output")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/json_output")

			bulks := ParseExportedFiles(logr.Discard(), "TestDef", tmMeta, input)
			bulkFile, err := bulks.Marshal()
			Expect(err).ToNot(HaveOccurred())

			ok, err := areEqualDocuments(bulkFile[0], output)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
	})

	Context("Parse bulk format", func() {
		It("should parse bulk with index and not add an testmachinery index", func() {
			input, err := os.ReadFile("./testdata/bulk_with_meta")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/bulk_with_meta")

			output, err := os.ReadFile("./testdata/bulk_with_meta_output")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/bulk_with_meta_output")

			bulks := ParseExportedFiles(logr.Discard(), "TestDef", tmMeta, input)
			bulkFile, err := bulks.Marshal()
			Expect(err).ToNot(HaveOccurred())

			ok, err := areEqualDocuments(bulkFile[0], output)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "Generated bulk document does not equal output document: %s", string(bulkFile[0]))
		})

		It("should parse bulk with index and add an testmachinery index", func() {
			input, err := os.ReadFile("./testdata/bulk_no_meta")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/bulk_no_meta")

			output, err := os.ReadFile("./testdata/bulk_no_meta_output")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/bulk_no_meta_output")

			bulks := ParseExportedFiles(logr.Discard(), "TestDef", tmMeta, input)
			bulkFile, err := bulks.Marshal()
			Expect(err).ToNot(HaveOccurred())

			ok, err := areEqualDocuments(bulkFile[0], output)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "Generated bulk document does not equal output document: %s", string(bulkFile[0]))
		})

		It("should parse a large bulk file", func() {
			input, err := os.ReadFile("./testdata/large_bulk")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/bulk_with_meta")

			bulks := ParseExportedFiles(logr.Discard(), "TestDef", tmMeta, input)
			Expect(len(bulks) != 0).To(BeTrue(), "Expect bulks to not be emtpy")

			_, err = bulks[0].Marshal()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func areEqualDocuments(input, output []byte) (bool, error) {
	inputScanner := bufio.NewScanner(bytes.NewReader(input))
	outputScanner := bufio.NewScanner(bytes.NewReader(output))
	for outputScanner.Scan() {
		inputScanner.Scan()
		out := outputScanner.Bytes()
		in := inputScanner.Bytes()

		if ok, err := areEqualJSON(out, in); !ok || err != nil {
			return ok, err
		}
	}
	if err := outputScanner.Err(); err != nil {
		return false, err
	}
	if err := inputScanner.Err(); err != nil {
		return false, err
	}

	return true, nil
}

func areEqualJSON(s1, s2 []byte) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal(s1, &o1)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 1 :: %s", err.Error())
	}
	err = json.Unmarshal(s2, &o2)
	if err != nil {
		return false, fmt.Errorf("Error mashalling string 2 :: %s", err.Error())
	}

	return reflect.DeepEqual(o1, o2), nil
}
