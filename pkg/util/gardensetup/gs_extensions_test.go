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

package gardensetup_test

import (
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util/gardensetup"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("gardensetup extensions util", func() {

	Context("Merge", func() {
		It("should merge two extensions with the same keys", func() {
			base := common.GSExtensions{
				"key1": common.GSExtensionConfig{Repository: "a1"},
				"key2": common.GSExtensionConfig{Repository: "a2"},
			}
			newVal := common.GSExtensions{
				"key1": common.GSExtensionConfig{Repository: "b1"},
				"key2": common.GSExtensionConfig{Repository: "b2"},
			}
			expected := common.GSExtensions{
				"key1": common.GSExtensionConfig{Repository: "b1"},
				"key2": common.GSExtensionConfig{Repository: "b2"},
			}

			res := gardensetup.MergeExtensions(base, newVal)
			Expect(res).To(Equal(expected))
		})

		It("should keep base key if their are not defined by the new value", func() {
			base := common.GSExtensions{
				"key1": common.GSExtensionConfig{Repository: "a1"},
				"key2": common.GSExtensionConfig{Repository: "a2"},
			}
			newVal := common.GSExtensions{
				"key1": common.GSExtensionConfig{Repository: "b1"},
			}
			expected := common.GSExtensions{
				"key1": common.GSExtensionConfig{Repository: "b1"},
				"key2": common.GSExtensionConfig{Repository: "a2"},
			}

			res := gardensetup.MergeExtensions(base, newVal)
			Expect(res).To(Equal(expected))
		})
		It("should add additional values defined by the new key to existing base keys", func() {
			base := common.GSExtensions{
				"key1": common.GSExtensionConfig{Repository: "a1"},
				"key2": common.GSExtensionConfig{Repository: "a2"},
			}
			newVal := common.GSExtensions{
				"key1": common.GSExtensionConfig{Repository: "b1"},
				"key3": common.GSExtensionConfig{Repository: "b1"},
			}
			expected := common.GSExtensions{
				"key1": common.GSExtensionConfig{Repository: "b1"},
				"key2": common.GSExtensionConfig{Repository: "a2"},
				"key3": common.GSExtensionConfig{Repository: "b1"},
			}

			res := gardensetup.MergeExtensions(base, newVal)
			Expect(res).To(Equal(expected))
		})
	})

	Context("parse flag", func() {
		It("should parse one extensions definition with a valid version", func() {
			f := "ext1=repo1:0.0.1"
			res, err := gardensetup.ParseFlag(f)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(common.GSExtensions{
				"ext1": common.GSExtensionConfig{Repository: "repo1", Tag: "0.0.1"},
			}))
		})

		It("should parse one extensions definition with a commit", func() {
			f := "ext1=repo1:000000000a000000b0000000000000c000000000"
			res, err := gardensetup.ParseFlag(f)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(common.GSExtensions{
				"ext1": common.GSExtensionConfig{Repository: "repo1", Commit: "000000000a000000b0000000000000c000000000", ImageTag: "000000000a000000b0000000000000c000000000"},
			}))
		})

		It("should parse one extensions definition with a branch", func() {
			f := "ext1=repo1:patch-1"
			res, err := gardensetup.ParseFlag(f)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(common.GSExtensions{
				"ext1": common.GSExtensionConfig{Repository: "repo1", Branch: "patch-1"},
			}))
		})

		It("should parse multiple extension definitions", func() {
			f := "ext1=repo1:0.0.1,ext2=repo2:0.0.0"
			res, err := gardensetup.ParseFlag(f)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(common.GSExtensions{
				"ext1": common.GSExtensionConfig{Repository: "repo1", Tag: "0.0.1"},
				"ext2": common.GSExtensionConfig{Repository: "repo2", Tag: "0.0.0"},
			}))
		})
	})
})
