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
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("util test", func() {

	Context("github url parsing", func() {
		It("should return the repo and owner of a repository", func() {
			ghUrl := "https://github.com/gardener/gardener-extensions.git"

			owner, repo, err := util.ParseRepoURLFromString(ghUrl)
			Expect(err).ToNot(HaveOccurred())
			Expect(repo).To(Equal("gardener-extensions"))
			Expect(owner).To(Equal("gardener"))
		})
	})

	DescribeTable("IsLastElementOfBucket",
		func(value int, expected bool) {
			Expect(util.IsLastElementOfBucket(value, 3)).To(Equal(expected))
		},
		Entry("0", 0, false),
		Entry("0", 1, false),
		Entry("2", 2, true),
		Entry("3", 3, false),
		Entry("4", 4, false),
		Entry("5", 5, true))
})
