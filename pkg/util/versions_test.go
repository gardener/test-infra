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
	"github.com/Masterminds/semver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/util"
)

var _ = Describe("versions util", func() {

	Context("get version from constraint", func() {
		It("should return the latest version", func() {
			versions := []*semver.Version{
				semver.MustParse("1.0.0"),
				semver.MustParse("1.2.2"),
				semver.MustParse("1.2.3"),
			}

			res, err := util.GetLatestVersionFromConstraint(versions, "*")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.String()).To(Equal("1.2.3"))
		})

		It("should return the latest version with latest constraint", func() {
			versions := []*semver.Version{
				semver.MustParse("1.0.0"),
				semver.MustParse("1.2.2"),
				semver.MustParse("1.2.3"),
			}

			res, err := util.GetLatestVersionFromConstraint(versions, "latest")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.String()).To(Equal("1.2.3"))
		})

		It("should return the latest version of a minor version", func() {
			versions := []*semver.Version{
				semver.MustParse("1.0.0"),
				semver.MustParse("1.2.2"),
				semver.MustParse("1.2.3"),
				semver.MustParse("1.3.5"),
			}

			res, err := util.GetLatestVersionFromConstraint(versions, "1.2.x")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.String()).To(Equal("1.2.3"))
		})

		It("should return the specific version of the constraint", func() {
			versions := []*semver.Version{
				semver.MustParse("0.0.9"),
				semver.MustParse("1.0.0"),
				semver.MustParse("1.2.2"),
				semver.MustParse("1.2.3"),
				semver.MustParse("1.3.5"),
			}

			res, err := util.GetLatestVersionFromConstraint(versions, "1.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.String()).To(Equal("1.0.0"))
		})

		It("should return the specific version of the constraint when a greater version is also defined", func() {
			versions := []*semver.Version{
				semver.MustParse("2.0.0"),
				semver.MustParse("1.0.0"),
				semver.MustParse("1.2.2"),
				semver.MustParse("1.2.3"),
				semver.MustParse("1.3.5"),
			}

			res, err := util.GetLatestVersionFromConstraint(versions, "1.0.0")
			Expect(err).ToNot(HaveOccurred())
			Expect(res.String()).To(Equal("1.0.0"))
		})
	})
})
