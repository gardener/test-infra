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

package bulk

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
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
			input, err := ioutil.ReadFile("./testdata/json")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/json")

			output, err := ioutil.ReadFile("./testdata/json_output")
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
			input, err := ioutil.ReadFile("./testdata/bulk_with_meta")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/bulk_with_meta")

			output, err := ioutil.ReadFile("./testdata/bulk_with_meta_output")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/bulk_with_meta_output")

			bulks := ParseExportedFiles(logr.Discard(), "TestDef", tmMeta, input)
			bulkFile, err := bulks.Marshal()
			Expect(err).ToNot(HaveOccurred())

			ok, err := areEqualDocuments(bulkFile[0], output)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "Generated bulk document does not equal output document: %s", string(bulkFile[0]))
		})

		It("should parse bulk with index and add an testmachinery index", func() {
			input, err := ioutil.ReadFile("./testdata/bulk_no_meta")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/bulk_no_meta")

			output, err := ioutil.ReadFile("./testdata/bulk_no_meta_output")
			Expect(err).ToNot(HaveOccurred(), "Cannot read json file from ./testdata/bulk_no_meta_output")

			bulks := ParseExportedFiles(logr.Discard(), "TestDef", tmMeta, input)
			bulkFile, err := bulks.Marshal()
			Expect(err).ToNot(HaveOccurred())

			ok, err := areEqualDocuments(bulkFile[0], output)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue(), "Generated bulk document does not equal output document: %s", string(bulkFile[0]))
		})

		It("should parse a large bulk file", func() {
			input, err := ioutil.ReadFile("./testdata/large_bulk")
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
