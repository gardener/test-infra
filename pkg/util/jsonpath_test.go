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

package util_test

import (
	"github.com/gardener/test-infra/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("jsonpath util", func() {

	Context("raw yaml jsonpath", func() {
		It("should get json path with dep of 2", func() {
			content := `
test:
  dep1:
    dep2: val
`
			var parsedRes string
			res, err := util.RawJSONPath([]byte(content), "test.dep1.dep2", &parsedRes)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(res)).To(Equal("\"val\""))
			Expect(parsedRes).To(Equal("val"))
		})

		It("should get json path and return struct", func() {
			content := `
test:
  dep1:
    dep2: val
`
			expected := map[string]interface{}{
				"dep2": "val",
			}
			var result map[string]interface{}
			_, err := util.RawJSONPath([]byte(content), "test.dep1", &result)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		})
	})

	Context("json path", func() {
		var (
			data interface{}
		)

		BeforeEach(func() {
			data = map[string]interface{}{
				"l1": map[string]interface{}{
					"l2": map[string]interface{}{
						"e1": 1,
						"e2": "string",
					},
				},
			}
		})

		It("Should return primitive value in third level", func() {
			res, err := util.JSONPath(data, "l1.l2.e1")
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(1))
		})

		It("Should return a map on the second level", func() {
			res, err := util.JSONPath(data, "l1.l2")
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(HaveKey("e1"))
			Expect(res).To(HaveKey("e2"))
		})

		It("Should error if the path is not defined", func() {
			_, err := util.JSONPath(data, "l1.l3")
			Expect(err).To(HaveOccurred())
		})

	})

})
