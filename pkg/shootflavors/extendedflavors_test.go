// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shootflavors

import (
	"context"

	"github.com/gardener/gardener-extension-provider-aws/pkg/apis/aws/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	mockclient "github.com/gardener/gardener/third_party/mock/controller-runtime/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/utils/ptr"
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
						Versions: MachineImageVersions(map[string][]string{"0.0.2": {"amd64"}, "0.0.1": {"amd64"}}),
					},
					{
						Name:     "test-os-2",
						Versions: MachineImageVersions(map[string][]string{"0.0.4": {"arm64"}, "0.0.3": {"arm64"}}),
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
				AdditionalAnnotations: map[string]string{"a": "b"},
				AdditionalLocations:   []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
				Provider:              common.CloudProviderGCP,
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
			Provider:              common.CloudProviderGCP,
			AdditionalAnnotations: map[string]string{"a": "b"},
			AdditionalLocations:   []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
			KubernetesVersion:     gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			Workers:               []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: ptr.To("amd64")}}},
		}))
		Expect(shoot.Get().ExtendedConfiguration).To(Equal(defaultExtendedCfg))
	})

	It("should return 1 shoot with worker pool having machine of CPU architecture arm64", func() {
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				AdditionalAnnotations: map[string]string{"a": "b"},
				AdditionalLocations:   []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
				Provider:              common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: ptr.To("arm64")}}},
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
			Provider:              common.CloudProviderGCP,
			AdditionalAnnotations: map[string]string{"a": "b"},
			AdditionalLocations:   []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
			KubernetesVersion:     gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			Workers:               []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: ptr.To("arm64")}}},
		}))
		Expect(shoot.Get().ExtendedConfiguration).To(Equal(defaultExtendedCfg))
	})

	It("should fail with invalid CPU architecture of machine in a worker pool", func() {
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				AdditionalAnnotations: map[string]string{"a": "b"},
				AdditionalLocations:   []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
				Provider:              common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: ptr.To("foo")}}},
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
										Version: ptr.To("latest"),
									},
								},
							},
							{
								Name: "wp1",
								Machine: gardencorev1beta1.Machine{
									Image: &gardencorev1beta1.ShootMachineImage{
										Name:    "test-os-2",
										Version: ptr.To("latest"),
									},
									Architecture: ptr.To("arm64"),
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

	It("should generate a providerConfig for workers on aws", func() {
		versionPattern := "latest"
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				Provider: common.CloudProviderAWS,
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
										Version: ptr.To("latest"),
									},
								},
							},
							{
								Name: "wp1",
								Machine: gardencorev1beta1.Machine{
									Image: &gardencorev1beta1.ShootMachineImage{
										Name:    "test-os-2",
										Version: ptr.To("latest"),
									},
									Architecture: ptr.To("arm64"),
								},
							},
						},
					},
				},
			},
		}}

		cloudprofile.Spec.Type = "aws"
		c.EXPECT().Get(gomock.Any(), client.ObjectKey{Name: "test-profile"}, gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, _ client.ObjectKey, obj *gardencorev1beta1.CloudProfile, _ ...client.GetOption) error {
			*obj = cloudprofile
			return nil
		})
		flavors, err := NewExtended(c, rawFlavors, "test-pref", false)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))

		Expect(*flavors.GetShoots()[0].Get().Workers[1].Machine.Image.Version).To(Equal("0.0.4"))
		Expect(*flavors.GetShoots()[0].Get().Workers[1].ProviderConfig).ToNot(BeNil())
		worker := v1alpha1.WorkerConfig{}
		err = json.Unmarshal(flavors.GetShoots()[0].Get().Workers[1].ProviderConfig.Raw, &worker)
		Expect(err).ToNot(HaveOccurred())
		Expect(*worker.InstanceMetadataOptions.HTTPTokens).To(Equal(v1alpha1.HTTPTokensRequired))
		Expect(*worker.InstanceMetadataOptions.HTTPPutResponseHopLimit).To(Equal(int64(2)))
		Expect(*flavors.GetShoots()[0].Get().Workers[0].Machine.Image.Version).To(Equal("0.0.2"))
		Expect(*flavors.GetShoots()[0].Get().Workers[0].ProviderConfig).ToNot(BeNil())
	})

	It("should generate a shoot with correct networkingType", func() {
		defaultExtendedCfg.NetworkingType = "calico"
		rawFlavors := []*common.ExtendedShootFlavor{{
			ExtendedConfiguration: defaultExtendedCfg,
			ShootFlavor: common.ShootFlavor{
				AdditionalAnnotations: map[string]string{"a": "b"},
				AdditionalLocations:   []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
				Provider:              common.CloudProviderGCP,
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
			Provider:              common.CloudProviderGCP,
			AdditionalAnnotations: map[string]string{"a": "b"},
			AdditionalLocations:   []common.AdditionalLocation{{Type: "git", Repo: "https:// github.com/gardener/gardener", Revision: "master"}},
			KubernetesVersion:     gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			Workers:               []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Architecture: ptr.To("amd64")}}},
		}))
		Expect(shoot.Get().ExtendedConfiguration).To(Equal(defaultExtendedCfg))
	})

	It("should generate a shoot with the correct controlPlaneFailureTolerance set", func() {
		const failureToleranceType = "zone"
		defaultExtendedCfg.ControlPlaneFailureTolerance = failureToleranceType
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
		Expect(shoot.Get().ControlPlaneFailureTolerance).To(Equal(failureToleranceType))
	})

	It("should generate a shoot with an empty controlPlaneFailureTolerance, if the value is not specified", func() {
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
		Expect(shoot.Get().ControlPlaneFailureTolerance).To(BeEmpty())
	})
})
