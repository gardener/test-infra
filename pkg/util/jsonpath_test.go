// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/util"
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
