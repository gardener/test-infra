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

package util_test

import (
	"github.com/gardener/test-infra/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("jsonpath util", func() {
	It("should get json path with dep of 2", func() {
		content := `
test:
  dep1:
    dep2: val
`
		var parsedRes string
		res, err := util.JSONPath([]byte(content), "test.dep1.dep2", &parsedRes)
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
		_, err := util.JSONPath([]byte(content), "test.dep1", &result)
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(expected))
	})
})
