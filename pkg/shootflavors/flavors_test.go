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
	"github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/test-infra/pkg/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("flavor test", func() {
	var (
		defaultMachine v1alpha1.Machine
	)
	BeforeEach(func() {
		defaultMachine = v1alpha1.Machine{
			Type:  "test-machine",
			Image: &v1alpha1.ShootMachineImage{Name: "coreos", Version: "0.0.1"},
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
					Versions: &[]v1alpha1.ExpirableVersion{
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
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.15"},
			},
		))
	})

	It("should return 2 gcp shoots with the specified 2 versions", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]v1alpha1.ExpirableVersion{
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
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.15"},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.14"},
			},
		))
	})

	It("should return 4 gcp shoots that are a combination of kubernetes version and worker pool config", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]v1alpha1.ExpirableVersion{
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
						WorkerPools: []v1alpha1.Worker{{Name: "wp1", Machine: defaultMachine}},
					},
					{
						WorkerPools: []v1alpha1.Worker{{Name: "wp2", Machine: defaultMachine}},
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
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.15"},
				Workers:           []v1alpha1.Worker{{Name: "wp1", Machine: defaultMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.14"},
				Workers:           []v1alpha1.Worker{{Name: "wp1", Machine: defaultMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.15"},
				Workers:           []v1alpha1.Worker{{Name: "wp2", Machine: defaultMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.14"},
				Workers:           []v1alpha1.Worker{{Name: "wp2", Machine: defaultMachine}},
			},
		))
	})

	It("should return 3 gcp shoots with old versions and one workerpool and 1 new version with multiple worker pools", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]v1alpha1.ExpirableVersion{
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
						WorkerPools: []v1alpha1.Worker{{Name: "wp1", Machine: defaultMachine}},
					},
				},
			},
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]v1alpha1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []v1alpha1.Worker{{Name: "wp1", Machine: defaultMachine}, {Name: "wp2", Machine: defaultMachine}},
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
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.14"},
				Workers:           []v1alpha1.Worker{{Name: "wp1", Machine: defaultMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.13"},
				Workers:           []v1alpha1.Worker{{Name: "wp1", Machine: defaultMachine}},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.15"},
				Workers:           []v1alpha1.Worker{{Name: "wp1", Machine: defaultMachine}, {Name: "wp2", Machine: defaultMachine}},
			},
		))
	})

	It("should return 2 gcp shoots and 1 aws with the specified k8s versions", func() {
		rawFlavors := []*common.ShootFlavor{
			{
				Provider: common.CloudProviderGCP,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]v1alpha1.ExpirableVersion{
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
					Versions: &[]v1alpha1.ExpirableVersion{
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
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.15"},
			},
			&common.Shoot{
				Provider:          common.CloudProviderGCP,
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.14"},
			},
			&common.Shoot{
				Provider:          common.CloudProviderAWS,
				KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.15"},
			},
		))
	})

	Context("used kubernetes version per cloudprovider", func() {
		It("should add one version for gcp to used versions", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]v1alpha1.ExpirableVersion{
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
			Expect(k8sVersions).To(HaveKeyWithValue(common.CloudProviderGCP, v1alpha1.KubernetesSettings{Versions: []v1alpha1.ExpirableVersion{{Version: "1.15"}}}))
		})

		It("should add different versions for gcp and aws", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]v1alpha1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
				},
				{
					Provider: common.CloudProviderAWS,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]v1alpha1.ExpirableVersion{
							{Version: "1.15"},
							{Version: "1.14"},
						},
					},
				},
			}
			flavors, err := New(rawFlavors)
			Expect(err).ToNot(HaveOccurred())

			k8sVersions := flavors.GetUsedKubernetesVersions()
			Expect(k8sVersions).To(HaveKeyWithValue(common.CloudProviderGCP, v1alpha1.KubernetesSettings{Versions: []v1alpha1.ExpirableVersion{{Version: "1.15"}}}))
			Expect(k8sVersions).To(HaveKeyWithValue(common.CloudProviderAWS, v1alpha1.KubernetesSettings{Versions: []v1alpha1.ExpirableVersion{{Version: "1.15"}, {Version: "1.14"}}}))
		})

		It("should add 2 unique versions from different flavors to the same cloudprovider", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]v1alpha1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
				},
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]v1alpha1.ExpirableVersion{
							{Version: "1.15"},
							{Version: "1.14"},
						},
					},
				},
			}
			flavors, err := New(rawFlavors)
			Expect(err).ToNot(HaveOccurred())

			k8sVersions := flavors.GetUsedKubernetesVersions()
			Expect(k8sVersions).To(HaveKeyWithValue(common.CloudProviderGCP, v1alpha1.KubernetesSettings{Versions: []v1alpha1.ExpirableVersion{{Version: "1.15"}, {Version: "1.14"}}}))
		})
	})

	Context("used machine images per cloudprovider", func() {
		It("should add one image with one version for gcp to used images", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]v1alpha1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
					Workers: []common.ShootWorkerFlavor{
						{
							WorkerPools: []v1alpha1.Worker{{Name: "wp1", Machine: defaultMachine}},
						},
					},
				},
			}
			flavors, err := New(rawFlavors)
			Expect(err).ToNot(HaveOccurred())

			images := flavors.GetUsedMachineImages()
			Expect(images).To(HaveKeyWithValue(common.CloudProviderGCP, []v1alpha1.MachineImage{{Name: "coreos", Versions: []v1alpha1.ExpirableVersion{{Version: "0.0.1"}}}}))
		})

		It("should add 2 image from different pools to gcp's used images", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]v1alpha1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
					Workers: []common.ShootWorkerFlavor{
						{
							WorkerPools: []v1alpha1.Worker{
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
			Expect(images).To(HaveKeyWithValue(common.CloudProviderGCP, []v1alpha1.MachineImage{
				{Name: "coreos", Versions: []v1alpha1.ExpirableVersion{{Version: "0.0.1"}}},
				{Name: "jeos", Versions: []v1alpha1.ExpirableVersion{{Version: "0.0.2"}}},
			}))
		})

		It("should add 2 unique images from different pools to gcp's used images", func() {
			rawFlavors := []*common.ShootFlavor{
				{
					Provider: common.CloudProviderGCP,
					KubernetesVersions: common.ShootKubernetesVersionFlavor{
						Versions: &[]v1alpha1.ExpirableVersion{
							{Version: "1.15"},
						},
					},
					Workers: []common.ShootWorkerFlavor{
						{
							WorkerPools: []v1alpha1.Worker{
								{Name: "wp1", Machine: defaultMachine},
								{Name: "wp2", Machine: newMachineImage("jeos", "0.0.2")},
							},
						},
						{
							WorkerPools: []v1alpha1.Worker{
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
			Expect(images).To(HaveKeyWithValue(common.CloudProviderGCP, []v1alpha1.MachineImage{
				{Name: "coreos", Versions: []v1alpha1.ExpirableVersion{{Version: "0.0.1"}}},
				{Name: "jeos", Versions: []v1alpha1.ExpirableVersion{{Version: "0.0.2"}}},
			}))
		})
	})
})

func newMachineImage(imageName, version string) v1alpha1.Machine {
	return v1alpha1.Machine{
		Type:  "test-machine",
		Image: &v1alpha1.ShootMachineImage{Name: imageName, Version: version},
	}
}
