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
	gardenv1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("kubernetes version util", func() {

	Context("get version", func() {
		var (
			cloudprofile gardenv1alpha1.CloudProfile
		)
		BeforeEach(func() {
			cloudprofile = gardenv1alpha1.CloudProfile{Spec: gardenv1alpha1.CloudProfileSpec{
				Kubernetes: gardenv1alpha1.KubernetesSettings{
					Versions: []gardenv1alpha1.ExpirableVersion{
						newExpirableVersion("1.15.2"),
						newExpirableVersion("1.15.1"),
						newExpirableVersion("1.14.6"),
						newExpirableVersion("1.14.5"),
						newExpirableVersion("1.13.5"),
					},
				},
			}}
		})

		It("should return only the specified version", func() {
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Versions: &[]gardenv1alpha1.ExpirableVersion{newExpirableVersion("1.0.0")},
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.0.0"),
			))
		})

		It("should return the latest version", func() {
			pattern := "latest"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.15.2"),
			))
		})

		It("should return all versions", func() {
			pattern := "*"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.15.2"),
				newExpirableVersion("1.15.1"),
				newExpirableVersion("1.14.6"),
				newExpirableVersion("1.14.5"),
				newExpirableVersion("1.13.5"),
			))
		})

		It("should return versions greater than 1.14.5", func() {
			pattern := ">1.14.5"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.15.2"),
				newExpirableVersion("1.15.1"),
				newExpirableVersion("1.14.6"),
			))
		})

		It("should return only latest patch versions greater than 1.14.5", func() {
			pattern := ">1.14.5"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.15.2"),
				newExpirableVersion("1.14.6"),
			))
		})
	})

	Context("previous kubernetes version", func() {
		It("should return 1.14.5 and 1.14.6", func() {
			cp := gardenv1alpha1.CloudProfile{Spec: gardenv1alpha1.CloudProfileSpec{
				Kubernetes: gardenv1alpha1.KubernetesSettings{
					Versions: []gardenv1alpha1.ExpirableVersion{
						newExpirableVersion("1.15.2"),
						newExpirableVersion("1.15.1"),
						newExpirableVersion("1.14.6"),
						newExpirableVersion("1.14.5"),
						newExpirableVersion("1.13.5"),
					},
				},
			}}

			m, p, err := util.GetPreviousKubernetesVersions(cp, newExpirableVersion("1.15.2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Version).To(Equal("1.14.5"))
			Expect(p.Version).To(Equal("1.14.6"))
		})

		It("should return 1.14.6 and 1.14.6 if no other versions are available", func() {
			cp := gardenv1alpha1.CloudProfile{Spec: gardenv1alpha1.CloudProfileSpec{
				Kubernetes: gardenv1alpha1.KubernetesSettings{
					Versions: []gardenv1alpha1.ExpirableVersion{
						newExpirableVersion("1.15.2"),
						newExpirableVersion("1.15.1"),
						newExpirableVersion("1.14.6"),
						newExpirableVersion("1.13.5"),
					},
				},
			}}

			m, p, err := util.GetPreviousKubernetesVersions(cp, newExpirableVersion("1.15.2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Version).To(Equal("1.14.6"))
			Expect(p.Version).To(Equal("1.14.6"))
		})

		It("should return 1.15.2 and 1.15.2 if no previous minor version is available", func() {
			cp := gardenv1alpha1.CloudProfile{Spec: gardenv1alpha1.CloudProfileSpec{
				Kubernetes: gardenv1alpha1.KubernetesSettings{
					Versions: []gardenv1alpha1.ExpirableVersion{
						newExpirableVersion("1.15.2"),
						newExpirableVersion("1.15.1"),
						newExpirableVersion("1.13.5"),
					},
				},
			}}

			m, p, err := util.GetPreviousKubernetesVersions(cp, newExpirableVersion("1.15.2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Version).To(Equal("1.15.2"))
			Expect(p.Version).To(Equal("1.15.2"))
		})

	})
})
