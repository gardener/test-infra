// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"path/filepath"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/shootflavors"
)

var _ = Describe("shoot templates", func() {

	const (
		DEFAULT_TESTDATA_DIR = "./testdata/default"
		SHOOT_TESTDATA_DIR   = "./testdata/shoot"
		GARDENER_KUBECONFIG  = "./testdata/test-kubeconfig.yaml"

		COMPONENT_TESTDATA_PATH = "../componentdescriptor/testdata/"
		ROOT_COMPONENT          = "root-component.yaml"
		REPOSITORY              = "repositories/ocm-repo-ctf"
	)

	var (
		ctx    context.Context
		shoots []*shootflavors.ExtendedFlavorInstance
	)

	BeforeEach(func() {
		ctx = context.Background()
		shoots = []*shootflavors.ExtendedFlavorInstance{
			shootflavors.NewExtendedFlavorInstance(&common.ExtendedShoot{
				Shoot: common.Shoot{
					Provider:              common.CloudProviderGCP,
					KubernetesVersion:     gardencorev1beta1.ExpirableVersion{Version: "1.15.2"},
					Workers:               []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Image: &gardencorev1beta1.ShootMachineImage{Name: "core-os"}}}},
					AdditionalAnnotations: map[string]string{"a": "b"},
					AdditionalLocations:   []common.AdditionalLocation{{Type: "git", Repo: "https://github.com/gardener/gardener", Revision: "1.2.3"}},
				},
				ExtendedShootConfiguration: common.ExtendedShootConfiguration{
					Name:         "test-name",
					Namespace:    "garden-it",
					Cloudprofile: gardencorev1beta1.CloudProfile{},
					ExtendedConfiguration: common.ExtendedConfiguration{
						ProjectName:      "test-proj",
						CloudprofileName: "test",
						SecretBinding:    "test-sb",
						Region:           "region-1",
						Zone:             "region-1-1",
					},
				},
			}),
		}
	})

	AfterEach(func() {
		ctx.Done()
	})

	Context("shoot", func() {
		It("should render the basic shoot chart with all its necessary parameters", func() {
			params := &Parameters{
				GardenKubeconfigPath:     GARDENER_KUBECONFIG,
				FlavoredTestrunChartPath: filepath.Join(SHOOT_TESTDATA_DIR, "basic"),
				ComponentDescriptorPath:  filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
				Repository:               filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
			}

			runs, err := RenderTestruns(ctx, logr.Discard(), params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(1))
			tr := runs[0].Testrun

			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.projectNamespace", "garden-it"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.cloudprovider", "gcp"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.cloudprofile", "test"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.secretBinding", "test-sb"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.region", "region-1"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.zone", "region-1-1"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.shootAnnotations", "a=b"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.k8sVersion", "1.15.2"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.k8sPrevPrePatchVersion", "1.15.2"))
			Expect(tr.Annotations).To(HaveKeyWithValue("shoot.k8sPrevPatchVersion", "1.15.2"))

			Expect(tr.Spec.LocationSets[0].Locations).To(ContainElement(v1beta1.TestLocation{
				Type:     "git",
				Repo:     "https://github.com/gardener/gardener",
				Revision: "1.2.3",
			}))
		})

		It("should render the basic shoot chart and write correct metadata", func() {
			params := &Parameters{
				GardenKubeconfigPath:     GARDENER_KUBECONFIG,
				FlavoredTestrunChartPath: filepath.Join(SHOOT_TESTDATA_DIR, "basic"),
				ComponentDescriptorPath:  filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
				Repository:               filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
				Landscape:                "test",
			}

			runs, err := RenderTestruns(ctx, logr.Discard(), params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(1))
			meta := runs[0].Metadata

			Expect(meta.Landscape).To(Equal("test"))
			Expect(meta.KubernetesVersion).To(Equal("1.15.2"))
			Expect(meta.CloudProvider).To(Equal("gcp"))
			Expect(meta.Region).To(Equal("region-1"))
			Expect(meta.Zone).To(Equal("region-1-1"))
			Expect(meta.Annotations).To(Equal(map[string]string{"a": "b"}))
			Expect(meta.OperatingSystem).To(Equal("core-os"))
		})

		It("should render the basic shoot chart and fetch all correct k8s versions", func() {
			params := &Parameters{
				GardenKubeconfigPath:     GARDENER_KUBECONFIG,
				FlavoredTestrunChartPath: filepath.Join(SHOOT_TESTDATA_DIR, "basic"),
				ComponentDescriptorPath:  filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
				Repository:               filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
			}
			shoots = []*shootflavors.ExtendedFlavorInstance{
				shootflavors.NewExtendedFlavorInstance(&common.ExtendedShoot{
					Shoot: common.Shoot{
						Provider:          common.CloudProviderGCP,
						KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.15.2"},
						Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Image: &gardencorev1beta1.ShootMachineImage{Name: "core-os"}}}},
					},
					ExtendedShootConfiguration: common.ExtendedShootConfiguration{
						Name:      "test-name",
						Namespace: "garden-it",
						Cloudprofile: gardencorev1beta1.CloudProfile{Spec: gardencorev1beta1.CloudProfileSpec{
							Kubernetes: gardencorev1beta1.KubernetesSettings{Versions: []gardencorev1beta1.ExpirableVersion{
								{Version: "1.15.2"},
								{Version: "1.14.1"},
								{Version: "1.14.0"},
								{Version: "1.13.8"},
							}},
						},
						},
						ExtendedConfiguration: common.ExtendedConfiguration{
							ProjectName:      "test-proj",
							CloudprofileName: "test",
							SecretBinding:    "test-sb",
							Region:           "region-1",
							Zone:             "region-1-1",
						},
					},
				}),
			}
			runs, err := RenderTestruns(ctx, logr.Discard(), params, shoots)
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
				GardenKubeconfigPath:     GARDENER_KUBECONFIG,
				FlavoredTestrunChartPath: filepath.Join(SHOOT_TESTDATA_DIR, "basic"),
				ComponentDescriptorPath:  filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
				Repository:               filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
			}
			shoots = append(shoots, shootflavors.NewExtendedFlavorInstance(&common.ExtendedShoot{
				Shoot: common.Shoot{
					Provider:          common.CloudProviderAWS,
					KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.16.2"},
					Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Image: &gardencorev1beta1.ShootMachineImage{Name: "suse-os"}}}},
				},
				ExtendedShootConfiguration: common.ExtendedShootConfiguration{
					Name:         "test-name",
					Namespace:    "garden-it",
					Cloudprofile: gardencorev1beta1.CloudProfile{},
					ExtendedConfiguration: common.ExtendedConfiguration{
						ProjectName:      "test-proj",
						CloudprofileName: "test",
						SecretBinding:    "test-sb-aws",
						Region:           "region1",
						Zone:             "region1c",
					},
				},
			}))
			runs, err := RenderTestruns(ctx, logr.Discard(), params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(2))
		})
	})

	Context("both", func() {
		It("should render the basic shoot chart and the default testrun", func() {
			params := &Parameters{
				GardenKubeconfigPath:     GARDENER_KUBECONFIG,
				FlavoredTestrunChartPath: filepath.Join(SHOOT_TESTDATA_DIR, "basic"),
				DefaultTestrunChartPath:  filepath.Join(DEFAULT_TESTDATA_DIR, "basic"),
				ComponentDescriptorPath:  filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
				Repository:               filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
			}

			runs, err := RenderTestruns(ctx, logr.Discard(), params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(2))
		})

		It("should render the basic shoot chart and the default testrun", func() {
			params := &Parameters{
				GardenKubeconfigPath:     GARDENER_KUBECONFIG,
				FlavoredTestrunChartPath: filepath.Join(SHOOT_TESTDATA_DIR, "basic"),
				DefaultTestrunChartPath:  filepath.Join(DEFAULT_TESTDATA_DIR, "basic"),
				ComponentDescriptorPath:  filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
				Repository:               filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
			}

			runs, err := RenderTestruns(ctx, logr.Discard(), params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(2))
		})

		It("should render 2 basic shoot charts and 1 default testrun", func() {
			params := &Parameters{
				GardenKubeconfigPath:     GARDENER_KUBECONFIG,
				FlavoredTestrunChartPath: filepath.Join(SHOOT_TESTDATA_DIR, "basic"),
				DefaultTestrunChartPath:  filepath.Join(DEFAULT_TESTDATA_DIR, "basic"),
				ComponentDescriptorPath:  filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
				Repository:               filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
			}
			shoots = append(shoots, shootflavors.NewExtendedFlavorInstance(&common.ExtendedShoot{
				Shoot: common.Shoot{
					Provider:          common.CloudProviderAWS,
					KubernetesVersion: gardencorev1beta1.ExpirableVersion{Version: "1.16.2"},
					Workers:           []gardencorev1beta1.Worker{{Name: "wp1", Machine: gardencorev1beta1.Machine{Image: &gardencorev1beta1.ShootMachineImage{Name: "suse-os"}}}},
				},
				ExtendedShootConfiguration: common.ExtendedShootConfiguration{
					Name:         "test-name",
					Namespace:    "garden-it",
					Cloudprofile: gardencorev1beta1.CloudProfile{},
					ExtendedConfiguration: common.ExtendedConfiguration{
						ProjectName:      "test-proj",
						CloudprofileName: "test",
						SecretBinding:    "test-sb-aws",
						Region:           "region1",
						Zone:             "region1c",
					},
				},
			}))
			runs, err := RenderTestruns(ctx, logr.Discard(), params, shoots)
			Expect(err).ToNot(HaveOccurred())
			Expect(runs.GetTestruns()).To(HaveLen(3))
		})
	})

	Context("rerender", func() {
		It("should rerender the basic shoot chart with different shoot name but with all other same values", func() {
			params := &Parameters{
				GardenKubeconfigPath:     GARDENER_KUBECONFIG,
				FlavoredTestrunChartPath: filepath.Join(SHOOT_TESTDATA_DIR, "basic"),
				ComponentDescriptorPath:  filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
				Repository:               filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
			}

			runs, err := RenderTestruns(ctx, logr.Discard(), params, shoots)
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

			rerenderedRun, err := runs[0].Rerenderer.Rerender(tr)
			Expect(err).ToNot(HaveOccurred())
			Expect(rerenderedRun.Testrun.Annotations).To(HaveKeyWithValue("shoot.projectNamespace", "garden-it"))
			Expect(rerenderedRun.Testrun.Annotations).To(HaveKeyWithValue("shoot.cloudprovider", "gcp"))
			Expect(rerenderedRun.Testrun.Annotations).To(HaveKeyWithValue("shoot.cloudprofile", "test"))
			Expect(rerenderedRun.Testrun.Annotations).To(HaveKeyWithValue("shoot.secretBinding", "test-sb"))
			Expect(rerenderedRun.Testrun.Annotations).To(HaveKeyWithValue("shoot.region", "region-1"))
			Expect(rerenderedRun.Testrun.Annotations).To(HaveKeyWithValue("shoot.zone", "region-1-1"))
			Expect(rerenderedRun.Testrun.Annotations).To(HaveKeyWithValue("shoot.k8sVersion", "1.15.2"))
			Expect(rerenderedRun.Testrun.Annotations).To(HaveKeyWithValue("shoot.k8sPrevPrePatchVersion", "1.15.2"))
			Expect(rerenderedRun.Testrun.Annotations).To(HaveKeyWithValue("shoot.k8sPrevPatchVersion", "1.15.2"))

			Expect(rerenderedRun.Testrun.Annotations["shoot.name"]).ToNot(Equal(tr.Annotations["shoot.name"]))
		})
	})

})
