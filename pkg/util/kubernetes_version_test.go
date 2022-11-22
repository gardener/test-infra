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
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util"
)

var _ = Describe("kubernetes version util", func() {

	Context("get version", func() {
		var (
			cloudprofile gardencorev1beta1.CloudProfile
		)
		BeforeEach(func() {
			cloudprofile = gardencorev1beta1.CloudProfile{Spec: gardencorev1beta1.CloudProfileSpec{
				Kubernetes: gardencorev1beta1.KubernetesSettings{
					Versions: []gardencorev1beta1.ExpirableVersion{
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
				Versions: &[]gardencorev1beta1.ExpirableVersion{newExpirableVersion("1.0.0")},
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

		It("should return all patch versions of (latest minor - 1) version", func() {
			pattern := "oneMinorBeforeLatest"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.14.5"),
				newExpirableVersion("1.14.6"),
			))
		})

		It("should return latest patch version of (latest minor - 1) version and filtering", func() {
			pattern := "oneMinorBeforeLatest"
			filter := true
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern:             &pattern,
				FilterPatchVersions: &filter,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.14.6"),
			))
		})

		It("should return latest patch version of (latest minor - 2) version and filtering", func() {
			pattern := "twoMinorBeforeLatest"
			filter := true
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern:             &pattern,
				FilterPatchVersions: &filter,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.13.5"),
			))
		})

		It("should not be possible to decrement beyond 0", func() {
			pattern := "twoMinorBeforeLatest"
			filter := true
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern:             &pattern,
				FilterPatchVersions: &filter,
			}

			cloudprofile = gardencorev1beta1.CloudProfile{Spec: gardencorev1beta1.CloudProfileSpec{
				Kubernetes: gardencorev1beta1.KubernetesSettings{
					Versions: []gardencorev1beta1.ExpirableVersion{
						newExpirableVersion("2.1.4"),
					},
				},
			}}
			_, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).To(HaveOccurred())
		})

		It("should return latest patch of old minor version omitting patch and `filterPatchVersions: true`", func() {
			pattern := "1.14"
			filter := true
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern:             &pattern,
				FilterPatchVersions: &filter,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.14.6"),
			))
		})

		It("should return latest patch of old minor version using '*' and `filterPatchVersions: true`", func() {
			pattern := "1.14.*"
			filter := true
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern:             &pattern,
				FilterPatchVersions: &filter,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.14.6"),
			))
		})

		It("should return latest patch of old minor version using 'X' and `filterPatchVersions: true`", func() {
			pattern := "1.14.X"
			filter := true
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern:             &pattern,
				FilterPatchVersions: &filter,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.14.6"),
			))
		})

		It("should return all patch versions of an old minor version omitting patch and `filterPatchVersions`", func() {
			pattern := "1.14"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.14.5"),
				newExpirableVersion("1.14.6"),
			))
		})

		It("should return all patch versions of an old minor version using `*` and omitting `filterPatchVersions`", func() {
			pattern := "1.14.*"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.14.5"),
				newExpirableVersion("1.14.6"),
			))
		})

		It("should return all patch versions of an old minor version using `X` and omitting `filterPatchVersions`", func() {
			pattern := "1.14.X"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.14.5"),
				newExpirableVersion("1.14.6"),
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

		It("should return something (real) greater than 1.14", func() {
			pattern := ">1.14"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.15.2"),
			))
		})

		It("should return versions in between", func() {
			pattern := ">1.13 <1.15"
			versionFlavor := common.ShootKubernetesVersionFlavor{
				Pattern: &pattern,
			}

			versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(versions).To(ConsistOf(
				newExpirableVersion("1.14.6"),
			))
		})

		Context("filter patch versions", func() {
			It("should filter patch versions although default is false", func() {
				pattern := "*"
				filter := true
				versionFlavor := common.ShootKubernetesVersionFlavor{
					Pattern:             &pattern,
					FilterPatchVersions: &filter,
				}

				versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(versions).To(ConsistOf(
					newExpirableVersion("1.15.2"),
					newExpirableVersion("1.14.6"),
					newExpirableVersion("1.13.5"),
				))
			})

			It("should not filter patch versions although default is true", func() {
				pattern := "*"
				filter := false
				versionFlavor := common.ShootKubernetesVersionFlavor{
					Pattern:             &pattern,
					FilterPatchVersions: &filter,
				}

				versions, err := util.GetK8sVersions(cloudprofile, versionFlavor, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(versions).To(ConsistOf(
					newExpirableVersion("1.15.2"),
					newExpirableVersion("1.15.1"),
					newExpirableVersion("1.14.6"),
					newExpirableVersion("1.14.5"),
					newExpirableVersion("1.13.5"),
				))
			})
		})
	})

	Context("previous kubernetes version", func() {
		It("should return 1.14.5 and 1.14.6", func() {
			cp := gardencorev1beta1.CloudProfile{Spec: gardencorev1beta1.CloudProfileSpec{
				Kubernetes: gardencorev1beta1.KubernetesSettings{
					Versions: []gardencorev1beta1.ExpirableVersion{
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
			cp := gardencorev1beta1.CloudProfile{Spec: gardencorev1beta1.CloudProfileSpec{
				Kubernetes: gardencorev1beta1.KubernetesSettings{
					Versions: []gardencorev1beta1.ExpirableVersion{
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
			cp := gardencorev1beta1.CloudProfile{Spec: gardencorev1beta1.CloudProfileSpec{
				Kubernetes: gardencorev1beta1.KubernetesSettings{
					Versions: []gardencorev1beta1.ExpirableVersion{
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

		It("should return 1.14.0 and 1.14.1", func() {
			cp := gardencorev1beta1.CloudProfile{Spec: gardencorev1beta1.CloudProfileSpec{
				Kubernetes: gardencorev1beta1.KubernetesSettings{
					Versions: []gardencorev1beta1.ExpirableVersion{
						newExpirableVersion("1.15.2"),
						newExpirableVersion("1.14.1"),
						newExpirableVersion("1.14.0"),
						newExpirableVersion("1.13.5"),
					},
				},
			}}

			m, p, err := util.GetPreviousKubernetesVersions(cp, newExpirableVersion("1.15.2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(m.Version).To(Equal("1.14.0"))
			Expect(p.Version).To(Equal("1.14.1"))
		})

	})
})
