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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/util"
)

var _ = Describe("Slack Table Post", func() {

	Context("SplitStrings", func() {
		It("Should return the text if the text is lower than the given max size", func() {
			data := "Lorem ipsum"
			res := util.SplitString(data, 20)
			Expect(res).To(HaveLen(1))
			Expect(res[0]).To(Equal(data))
		})

		It("Should split by newline", func() {
			data := "Lor\nem\nipsum\ndolor"
			res := util.SplitString(data, 9)
			Expect(res).To(HaveLen(3))
			Expect(res[0]).To(Equal("Lor\nem\n"))
			Expect(res[1]).To(Equal("ipsum\n"))
			Expect(res[2]).To(Equal("dolor"))
		})
	})

})
