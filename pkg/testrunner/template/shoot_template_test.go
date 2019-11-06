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

package template

import (
	"github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/test-infra/pkg/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("shoot templates", func() {

	var (
		shoots []*common.ExtendedShoot
	)

	BeforeEach(func() {
		shoots = []*common.ExtendedShoot{
			{
				Shoot: common.Shoot{
					Provider:          common.CloudProviderGCP,
					KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.15.2"},
					Workers:           []v1alpha1.Worker{{Name: "wp1", Machine: v1alpha1.Machine{Image: &v1alpha1.ShootMachineImage{Name: "core-os"}}}},
				},
				ExtendedShootConfiguration: common.ExtendedShootConfiguration{
					Name:         "test-name",
					Namespace:    "garden-it",
					Cloudprofile: v1alpha1.CloudProfile{},
					ExtendedConfiguration: common.ExtendedConfiguration{
						ProjectName:      "test-proj",
						CloudprofileName: "test",
						SecretBinding:    "test-sb",
						Region:           "region-1",
						Zone:             "region-1-1",
					},
				},
			},
		}
	})

	Context("shoot", func() {
		It("should render the basic shoot chart with all its necessary parameters", func() {
			params := &Parameters{
				GardenKubeconfigPath:    gardenerKubeconfig,
				ShootTestrunChartPath:   filepath.Join(shootTestdataDir, "basic"),
				ComponentDescriptorPath: componentDescriptorPath,
			}

			runs, err := RenderTestruns(log.NullLogger{}, params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(1))
			tr := runs[0].Testrun

			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.projectNamespace", "garden-it"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.cloudprovider", "gcp"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.cloudprofile", "test"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.secretBinding", "test-sb"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.region", "region-1"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.zone", "region-1-1"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.k8sVersion", "1.15.2"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.k8sPrevPrePatchVersion", "1.15.2"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.k8sPrevPatchVersion", "1.15.2"))
		})

		It("should render the basic shoot chart and write correct metadata", func() {
			params := &Parameters{
				GardenKubeconfigPath:    gardenerKubeconfig,
				ShootTestrunChartPath:   filepath.Join(shootTestdataDir, "basic"),
				ComponentDescriptorPath: componentDescriptorPath,
				Landscape:               "test",
			}

			runs, err := RenderTestruns(log.NullLogger{}, params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(1))
			meta := runs[0].Metadata

			Expect(meta.Landscape).To(Equal("test"))
			Expect(meta.KubernetesVersion).To(Equal("1.15.2"))
			Expect(meta.CloudProvider).To(Equal("gcp"))
			Expect(meta.Region).To(Equal("region-1"))
			Expect(meta.Zone).To(Equal("region-1-1"))
			Expect(meta.OperatingSystem).To(Equal("core-os"))
		})

		It("should render the basic shoot chart and fetch all correct k8s versions", func() {
			params := &Parameters{
				GardenKubeconfigPath:    gardenerKubeconfig,
				ShootTestrunChartPath:   filepath.Join(shootTestdataDir, "basic"),
				ComponentDescriptorPath: componentDescriptorPath,
			}
			shoots[0].Cloudprofile = v1alpha1.CloudProfile{Spec: v1alpha1.CloudProfileSpec{
				Kubernetes: v1alpha1.KubernetesSettings{Versions: []v1alpha1.ExpirableVersion{
					{Version: "1.15.2"},
					{Version: "1.14.1"},
					{Version: "1.14.0"},
					{Version: "1.13.8"},
				}},
			}}
			runs, err := RenderTestruns(log.NullLogger{}, params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(1))
			tr := runs[0].Testrun
			meta := runs[0].Metadata

			Expect(meta.KubernetesVersion).To(Equal("1.15.2"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.k8sVersion", "1.15.2"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.k8sPrevPrePatchVersion", "1.14.0"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.k8sPrevPatchVersion", "1.14.1"))
		})

		It("should render 2 basic shoot charts", func() {
			params := &Parameters{
				GardenKubeconfigPath:    gardenerKubeconfig,
				ShootTestrunChartPath:   filepath.Join(shootTestdataDir, "basic"),
				ComponentDescriptorPath: componentDescriptorPath,
			}
			shoots = append(shoots, &common.ExtendedShoot{
				Shoot: common.Shoot{
					Provider:          common.CloudProviderAWS,
					KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.16.2"},
					Workers:           []v1alpha1.Worker{{Name: "wp1", Machine: v1alpha1.Machine{Image: &v1alpha1.ShootMachineImage{Name: "suse-os"}}}},
				},
				ExtendedShootConfiguration: common.ExtendedShootConfiguration{
					Name:         "test-name",
					Namespace:    "garden-it",
					Cloudprofile: v1alpha1.CloudProfile{},
					ExtendedConfiguration: common.ExtendedConfiguration{
						ProjectName:      "test-proj",
						CloudprofileName: "test",
						SecretBinding:    "test-sb-aws",
						Region:           "region1",
						Zone:             "region1c",
					},
				},
			})
			runs, err := RenderTestruns(log.NullLogger{}, params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(2))
		})
	})

	Context("both", func() {
		It("should render the basic shoot chart and the default testrun", func() {
			params := &Parameters{
				GardenKubeconfigPath:    gardenerKubeconfig,
				ShootTestrunChartPath:   filepath.Join(shootTestdataDir, "basic"),
				TestrunChartPath:        filepath.Join(defaultTestdataDir, "basic"),
				ComponentDescriptorPath: componentDescriptorPath,
			}

			runs, err := RenderTestruns(log.NullLogger{}, params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(2))
		})

		It("should render 2 basic shoot charts and 1 default testrun", func() {
			params := &Parameters{
				GardenKubeconfigPath:    gardenerKubeconfig,
				ShootTestrunChartPath:   filepath.Join(shootTestdataDir, "basic"),
				TestrunChartPath:        filepath.Join(defaultTestdataDir, "basic"),
				ComponentDescriptorPath: componentDescriptorPath,
			}
			shoots = append(shoots, &common.ExtendedShoot{
				Shoot: common.Shoot{
					Provider:          common.CloudProviderAWS,
					KubernetesVersion: v1alpha1.ExpirableVersion{Version: "1.16.2"},
					Workers:           []v1alpha1.Worker{{Name: "wp1", Machine: v1alpha1.Machine{Image: &v1alpha1.ShootMachineImage{Name: "suse-os"}}}},
				},
				ExtendedShootConfiguration: common.ExtendedShootConfiguration{
					Name:         "test-name",
					Namespace:    "garden-it",
					Cloudprofile: v1alpha1.CloudProfile{},
					ExtendedConfiguration: common.ExtendedConfiguration{
						ProjectName:      "test-proj",
						CloudprofileName: "test",
						SecretBinding:    "test-sb-aws",
						Region:           "region1",
						Zone:             "region1c",
					},
				},
			})
			runs, err := RenderTestruns(log.NullLogger{}, params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(3))
		})
	})

})
