// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shootflavors

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"

	"github.com/gardener/test-infra/pkg/common"
)

var _ = Describe("flavor test", func() {
	var (
		defaultMachine, defaultExpectedMachine gardencorev1beta1.Machine
	)
	BeforeEach(func() {
		defaultMachine = gardencorev1beta1.Machine{
			Type:  "test-machine",
			Image: &gardencorev1beta1.ShootMachineImage{Name: "coreos", Version: ptr.To("0.0.1")},
		}

		defaultExpectedMachine = gardencorev1beta1.Machine{
			Type:         "test-machine",
			Image:        &gardencorev1beta1.ShootMachineImage{Name: "coreos", Version: ptr.To("0.0.1")},
			Architecture: ptr.To("amd64"),
		}

	})
	It("should return no shoots if no flavors are defined", func() {
		rawFlavors := []*common.ShootFlavor{}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(0))
	})

	It("should error if a kubernetes pattern is defined", func() {
		pattern := "latest"
		rawFlavors := []*common.ShootFlavor{
			{
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Pattern: &pattern,
				},
			},
		}
		_, err := New(rawFlavors)
		Expect(err).To(HaveOccurred())
	})

	It("should return one shoot without a worker config", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			},
		))
	})

	It("should return one shoot with additional annotations", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider:              common.CloudProviderGCP,
				AdditionalAnnotations: map[string]string{"x": "y"},
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:              common.CloudProviderGCP,
				AdditionalAnnotations: map[string]string{"x": "y"},
				KubernetesVersion:     gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			},
		))
	})

	It("should return one shoot with additional locations", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider:            common.CloudProviderGCP,
				AdditionalLocations: []common.AdditionalLocation{{Type: "git", Repo: "github.com/org/name", Revision: "1.2.3"}},
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:            common.CloudProviderGCP,
				AdditionalLocations: []common.AdditionalLocation{{Type: "git", Repo: "github.com/org/name", Revision: "1.2.3"}},
				KubernetesVersion:   gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			},
		))
	})

	It("should return one shoot with worker pool machine architecture arm64", func() {
		machine := defaultMachine
		machine.Architecture = ptr.To("arm64")

		rawFlavors := []*common.ShootFlavor{
			{
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
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: machine}},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: machine}},
			},
		))
	})

	It("should fail with an inavlid architecture type", func() {
		machine := defaultMachine
		machine.Architecture = ptr.To("foo")

		rawFlavors := []*common.ShootFlavor{
			{
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
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: machine}},
					},
				},
			},
		}
		_, err := New(rawFlavors)
		Expect(err).To(HaveOccurred())
	})

	It("should fail with an incomplete additional location missing 'repo'", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider:            common.CloudProviderGCP,
				AdditionalLocations: []common.AdditionalLocation{{Type: "git", Revision: "1.2.3"}},
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
			},
		}
		_, err := New(rawFlavors)
		Expect(err).To(HaveOccurred())
	})

	It("should fail with an incomplete additional location missing 'revision'", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider:            common.CloudProviderGCP,
				AdditionalLocations: []common.AdditionalLocation{{Type: "git", Repo: "github.com/org/name"}},
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
			},
		}
		_, err := New(rawFlavors)
		Expect(err).To(HaveOccurred())
	})

	It("should fail with an incomplete additional location missing 'type'", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider:            common.CloudProviderGCP,
				AdditionalLocations: []common.AdditionalLocation{{Repo: "github.com/org/name", Revision: "1.2.3"}},
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
			},
		}
		_, err := New(rawFlavors)
		Expect(err).To(HaveOccurred())
	})

	It("should fail with an incomplete additional location missing multiple fields", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider:            common.CloudProviderGCP,
				AdditionalLocations: []common.AdditionalLocation{{}},
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
			},
		}
		_, err := New(rawFlavors)
		Expect(err).To(HaveOccurred())
	})

	It("should return 2 gcp shoots with the specified 2 versions", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
						{
							Version: "1.14",
						},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(2))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.14"},
			},
		))
	})

	It("should return 4 gcp shoots that are a combination of kubernetes version and worker pool config", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
						{
							Version: "1.14",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultMachine}},
					},
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp2", Machine: defaultMachine}},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(4))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultExpectedMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.14"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultExpectedMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp2", Machine: defaultExpectedMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.14"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp2", Machine: defaultExpectedMachine}},
			},
		))
	})

	It("should return 3 gcp shoots with old versions and one workerpool and 1 new version with multiple worker pools", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.14",
						},
						{
							Version: "1.13",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultMachine}},
					},
				},
			},
			{
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
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultMachine}, {Name: "wp2", Machine: defaultMachine}},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(3))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.14"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultExpectedMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.13"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultExpectedMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultExpectedMachine}, {Name: "wp2", Machine: defaultExpectedMachine}},
			},
		))
	})

	It("should return 1 gcp shoots with mulitple worker pool of different machine CPU architecture", func() {
		machine := defaultMachine
		machine.Architecture = ptr.To("arm64")
		rawFlavors := []*common.ShootFlavor{
			{
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
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultMachine}, {Name: "wp2", Machine: machine}},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(1))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultExpectedMachine}, {Name: "wp2", Machine: machine}},
			},
		))
	})

	It("should return 2 gcp shoots with specified CPU architecture of machine in a worker pool", func() {
		machine := defaultMachine
		machine.Architecture = ptr.To("arm64")
		rawFlavors := []*common.ShootFlavor{
			{
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
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultMachine}},
					},
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp2", Machine: machine}},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(2))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultExpectedMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
				Workers:           []gardencorev1beta1.Worker{{Name: "wp2", Machine: machine}},
			},
		))
	})

	It("should return 2 gcp shoots and 1 aws with the specified k8s versions", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
						{
							Version: "1.14",
						},
					},
				},
			},
			{
				Provider: common.CloudProviderAWS,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
			},
		}
		flavors, err := New(rawFlavors)
		Expect(err).ToNot(HaveOccurred())
		Expect(flavors.GetShoots()).To(HaveLen(3))
		Expect(flavors.GetShoots()).To(ConsistOf(
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.14"},
			},
			&common.Shoot{
				Provider:          common.CloudProviderAWS,
				KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15"},
			},
		))
	})

	Context("used kubernetes version per cloudprovider", func() {
		It("should add one version for gcp to used versions", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]gardencorev1beta1.ExpirableVersion{
							{
								Version: "1.15",
							},
						},
					},
				},
			}
			flavors, err := New(rawFlavors)
			Expect(err).ToNot(HaveOccurred())

			k8sVersions := flavors.GetUsedKubernetesVersions()
			Expect(k8sVersions).To(HaveKeyWithValue(common.CloudProviderGCP, gardencorev1beta1.KubernetesSettings{Versions: []gardencorev1beta1.ExpirableVersion{{Version: "1.15"}}}))
		})

		It("should add different versions for gcp and aws", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]gardencorev1beta1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
				},
				{
					Provider: common.CloudProviderAWS,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]gardencorev1beta1.ExpirableVersion{
							{Version: "1.15"},
							{Version: "1.14"},
						},
					},
				},
			}
			flavors, err := New(rawFlavors)
			Expect(err).ToNot(HaveOccurred())

			k8sVersions := flavors.GetUsedKubernetesVersions()
			Expect(k8sVersions).To(HaveKeyWithValue(common.CloudProviderGCP, gardencorev1beta1.KubernetesSettings{Versions: []gardencorev1beta1.ExpirableVersion{{Version: "1.15"}}}))
			Expect(k8sVersions).To(HaveKeyWithValue(common.CloudProviderAWS, gardencorev1beta1.KubernetesSettings{Versions: []gardencorev1beta1.ExpirableVersion{{Version: "1.15"}, {Version: "1.14"}}}))
		})

		It("should add 2 unique versions from different flavors to the same cloudprovider", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]gardencorev1beta1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
				},
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]gardencorev1beta1.ExpirableVersion{
							{Version: "1.15"},
							{Version: "1.14"},
						},
					},
				},
			}
			flavors, err := New(rawFlavors)
			Expect(err).ToNot(HaveOccurred())

			k8sVersions := flavors.GetUsedKubernetesVersions()
			Expect(k8sVersions).To(HaveKeyWithValue(common.CloudProviderGCP, gardencorev1beta1.KubernetesSettings{Versions: []gardencorev1beta1.ExpirableVersion{{Version: "1.15"}, {Version: "1.14"}}}))
		})
	})

	Context("used machine images per cloudprovider", func() {
		It("should add one image with one version for gcp to used images", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]gardencorev1beta1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
					Workers: []common.ShootWorkerFlavor{
						{
							WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1", Machine: defaultMachine}},
						},
					},
				},
			}
			flavors, err := New(rawFlavors)
			Expect(err).ToNot(HaveOccurred())

			images := flavors.GetUsedMachineImages()
			Expect(images).To(HaveKeyWithValue(common.CloudProviderGCP, []gardencorev1beta1.MachineImage{{
				Name:     "coreos",
				Versions: MachineImageVersions(map[string][]string{"0.0.1": {"amd64"}}),
			}}))
		})

		It("should add 2 image from different pools to gcp's used images", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]gardencorev1beta1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
					Workers: []common.ShootWorkerFlavor{
						{
							WorkerPools: []gardencorev1beta1.Worker{
								{Name: "wp1", Machine: defaultMachine},
								{Name: "wp2", Machine: newMachineImage("jeos", "0.0.2")},
							},
						},
					},
				},
			}
			flavors, err := New(rawFlavors)
			Expect(err).ToNot(HaveOccurred())

			images := flavors.GetUsedMachineImages()
			Expect(images).To(HaveKeyWithValue(common.CloudProviderGCP, []gardencorev1beta1.MachineImage{
				{Name: "coreos", Versions: MachineImageVersions(map[string][]string{"0.0.1": {"amd64"}})},
				{Name: "jeos", Versions: MachineImageVersions(map[string][]string{"0.0.2": {"amd64"}})},
			}))
		})

		It("should add 2 unique images from different pools to gcp's used images", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]gardencorev1beta1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
					Workers: []common.ShootWorkerFlavor{
						{
							WorkerPools: []gardencorev1beta1.Worker{
								{Name: "wp1", Machine: defaultMachine},
								{Name: "wp2", Machine: newMachineImage("jeos", "0.0.2")},
							},
						},
						{
							WorkerPools: []gardencorev1beta1.Worker{
								{Name: "wp1", Machine: defaultMachine},
								{Name: "wp2", Machine: newMachineImage("jeos", "0.0.2")},
							},
						},
					},
				},
			}
			flavors, err := New(rawFlavors)
			Expect(err).ToNot(HaveOccurred())

			images := flavors.GetUsedMachineImages()
			Expect(images).To(HaveKeyWithValue(common.CloudProviderGCP, []gardencorev1beta1.MachineImage{
				{Name: "coreos", Versions: MachineImageVersions(map[string][]string{"0.0.1": {"amd64"}})},
				{Name: "jeos", Versions: MachineImageVersions(map[string][]string{"0.0.2": {"amd64"}})},
			}))
		})
	})
})

func newMachineImage(imageName, version string) gardencorev1beta1.Machine {
	return gardencorev1beta1.Machine{
		Type:  "test-machine",
		Image: &gardencorev1beta1.ShootMachineImage{Name: imageName, Version: &version},
	}
}
