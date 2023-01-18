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

package shootflavors

import (
	"context"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/common"
)

var _ = Describe("extended flavor test", func() {
	var (
		ctrl *gomock.Controller
		c    *mockclient.MockClient

		defaultExtendedCfg common.ExtendedConfiguration
		cloudprofile       gardencorev1beta1.CloudProfile
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)

		defaultExtendedCfg = common.ExtendedConfiguration{
			ProjectName:      "test",
			CloudprofileName: "test-profile",
			SecretBinding:    "sb-test",
			Region:           "test-region",
			Zone:             "test-zone",
		}
		cloudprofile = gardencorev1beta1.CloudProfile{
			Spec: gardencorev1beta1.CloudProfileSpec{
				Kubernetes: gardencorev1beta1.KubernetesSettings{
					Versions: []gardencorev1beta1.ExpirableVersion{
						{Version: "1.16.1"},
						{Version: "1.15.2"},
						{Version: "1.15.1"},
						{Version: "1.14.3"},
						{Version: "1.13.10"},
					},
				},
				MachineImages: []gardencorev1beta1.MachineImage{
					{
						Name:     "test-os",
						Versions: MachineImageVersions(map[string][]string{"0.0.2": []string{"amd64"}, "0.0.1": []string{"amd64"}}),
					},
					{
						Name:     "test-os-2",
						Versions: MachineImageVersions(map[string][]string{"0.0.4": []string{"arm64"}, "0.0.3": []string{"arm64"}}),
					},
				},
			},
		}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should return no shoots if no flavors are defined", func() {
		rawFlavors := []*common.ExtendedShootFlavor{}
		flavors, err := NewExtended(c, rawFlavors, "", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(0))
	})

	It("should return 1 shoot", func() {
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				AllowPrivilegedContainers: pointer.Bool(true),
				AdditionalAnnotations:     map[string]string{"a": "b"},
				AdditionalLocations:       []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
				Provider:                  common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1"}},
					},
				},
			},
		}}

		c.EXPECT().Get(gomock.Any(), client.ObjectKey{Name: "test-profile"}, gomock.Any()).Times(1)
		flavors, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))

		shoot := flavors.GetShoots()[0]
		Expect(shoot.Get().Shoot).To(Equal(common.Shoot{
			Provider:                  common.CloudProviderGCP,
			AllowPrivilegedContainers: pointer.Bool(true),
			AdditionalAnnotations:     map[string]string{"a": "b"},
			AdditionalLocations:       []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
			KubernetesVersion:         gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			Workers:                   []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: pointer.String("amd64")}}},
		}))
		Expect(shoot.Get().ExtendedConfiguration).To(Equal(defaultExtendedCfg))
	})

	It("should return 1 shoot with worker pool having machine of CPU architecture arm64", func() {
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				AllowPrivilegedContainers: pointer.Bool(true),
				AdditionalAnnotations:     map[string]string{"a": "b"},
				AdditionalLocations:       []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
				Provider:                  common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: pointer.String("arm64")}}},
					},
				},
			},
		}}

		c.EXPECT().Get(gomock.Any(), client.ObjectKey{Name: "test-profile"}, gomock.Any()).Times(1)
		flavors, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))

		shoot := flavors.GetShoots()[0]
		Expect(shoot.Get().Shoot).To(Equal(common.Shoot{
			Provider:                  common.CloudProviderGCP,
			AllowPrivilegedContainers: pointer.Bool(true),
			AdditionalAnnotations:     map[string]string{"a": "b"},
			AdditionalLocations:       []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
			KubernetesVersion:         gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			Workers:                   []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: pointer.String("arm64")}}},
		}))
		Expect(shoot.Get().ExtendedConfiguration).To(Equal(defaultExtendedCfg))
	})

	It("should fail with invalid CPU architecture of machine in a worker pool", func() {
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				AllowPrivilegedContainers: pointer.Bool(true),
				AdditionalAnnotations:     map[string]string{"a": "b"},
				AdditionalLocations:       []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
				Provider:                  common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: pointer.String("foo")}}},
					},
				},
			},
		}}

		_, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).To(HaveOccurred())
	})

	It("should select the correct 3 versions", func() {
		versionPattern := ">=1.14 <= 1.15"
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Pattern: &versionPattern,
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1"}},
					},
				},
			},
		}}

		c.EXPECT().Get(gomock.Any(), client.ObjectKey{Name: "test-profile"}, gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *gardencorev1beta1.CloudProfile, _ ...client.GetOption) error {
			*obj = cloudprofile
			return nil
		})
		flavors, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(3))

		for _, shoot := range flavors.GetShoots() {
			Expect(shoot.Get().Shoot.KubernetesVersion.Version).To(Or(Equal("1.14.3"), Equal("1.15.1"), Equal("1.15.2")))
			Expect(shoot.Get().ExtendedConfiguration).To(Equal(defaultExtendedCfg))
		}
	})

	It("should add a prefix to the shoot name", func() {
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1"}},
					},
				},
			},
		}}

		c.EXPECT().Get(gomock.Any(), client.ObjectKey{Name: "test-profile"}, gomock.Any()).Times(1)
		flavors, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))

		shoot := flavors.GetShoots()[0]
		Expect(shoot.Get().Name).To(HavePrefix("test-pref"))
	})

	It("should generate a shoot with the latest kubernetes version from the cloudprofile", func() {
		versionPattern := "latest"
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Pattern: &versionPattern,
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1"}},
					},
				},
			},
		}}

		c.EXPECT().Get(gomock.Any(), client.ObjectKey{Name: "test-profile"}, gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *gardencorev1beta1.CloudProfile, _ ...client.GetOption) error {
			*obj = cloudprofile
			return nil
		})
		flavors, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))

		shoot := flavors.GetShoots()[0]
		Expect(shoot.Get().Shoot.KubernetesVersion.Version).To(Equal("1.16.1"))
		Expect(shoot.Get().ExtendedConfiguration).To(Equal(defaultExtendedCfg))
	})

	It("should generate a shoot for every kubernetes version from the cloudprofile", func() {
		versionPattern := "*"
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Pattern: &versionPattern,
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1"}},
					},
				},
			},
		}}

		c.EXPECT().Get(gomock.Any(), client.ObjectKey{Name: "test-profile"}, gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *gardencorev1beta1.CloudProfile, _ ...client.GetOption) error {
			*obj = cloudprofile
			return nil
		})
		flavors, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(5))
	})

	It("should generate a shoot with the latest machine image version from the cloudprofile", func() {
		versionPattern := "latest"
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Pattern: &versionPattern,
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{
							{
								Name: "wp1",
								Machine: gardencorev1beta1.Machine{
									Image: &gardencorev1beta1.ShootMachineImage{
										Name:    "test-os",
										Version: pointer.String("latest"),
									},
								},
							},
							{
								Name: "wp1",
								Machine: gardencorev1beta1.Machine{
									Image: &gardencorev1beta1.ShootMachineImage{
										Name:    "test-os-2",
										Version: pointer.String("latest"),
									},
									Architecture: pointer.String("arm64"),
								},
							},
						},
					},
				},
			},
		}}

		c.EXPECT().Get(gomock.Any(), client.ObjectKey{Name: "test-profile"}, gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *gardencorev1beta1.CloudProfile, _ ...client.GetOption) error {
			*obj = cloudprofile
			return nil
		})
		flavors, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))

		Expect(*flavors.GetShoots()[0].Get().Workers[0].Machine.Image.Version).To(Equal("0.0.2"))
		Expect(*flavors.GetShoots()[0].Get().Workers[1].Machine.Image.Version).To(Equal("0.0.4"))
	})

	It("should generate a shoot with correct networkingType", func() {
		defaultExtendedCfg.NetworkingType = "calico"
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				AllowPrivilegedContainers: pointer.Bool(true),
				AdditionalAnnotations:     map[string]string{"a": "b"},
				AdditionalLocations:       []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
				Provider:                  common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1"}},
					},
				},
			},
		}}

		c.EXPECT().Get(gomock.Any(), client.ObjectKey{Name: "test-profile"}, gomock.Any()).Times(1)
		flavors, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))

		shoot := flavors.GetShoots()[0]
		Expect(shoot.Get().Shoot).To(Equal(common.Shoot{
			Provider:                  common.CloudProviderGCP,
			AllowPrivilegedContainers: pointer.Bool(true),
			AdditionalAnnotations:     map[string]string{"a": "b"},
			AdditionalLocations:       []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
			KubernetesVersion:         gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			Workers:                   []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: pointer.String("amd64")}}},
		}))
		Expect(shoot.Get().ExtendedConfiguration).To(Equal(defaultExtendedCfg))
	})
})
