package util

import (
	"encoding/json"
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	"github.com/gardener/test-infra/pkg/common"
)

const (
	imageName  = "gardenlinux"
	arch_amd64 = "amd64"
	arch_arm64 = "arm64"
)

// regionFormatProviderConfig builds a RawExtension in the per-region format,
// emitting one region entry per architecture per version.
//
// Legacy: remove once migration to capability flavors is complete.
func regionFormatProviderConfig(imageName string, archsByVersion map[string][]string) *runtime.RawExtension {
	versions := make([]providerConfigMachineImageVersion, 0, len(archsByVersion))
	for version, archs := range archsByVersion {
		regions := make([]providerConfigRegion, 0, len(archs))
		for _, a := range archs {
			regions = append(regions, providerConfigRegion{Architecture: a})
		}
		versions = append(versions, providerConfigMachineImageVersion{
			Version: version,
			Regions: regions,
		})
	}
	cfg := providerConfigMachineImages{
		MachineImages: []providerConfigMachineImage{
			{Name: imageName, Versions: versions},
		},
	}
	raw, err := json.Marshal(cfg)
	Expect(err).ToNot(HaveOccurred())
	return &runtime.RawExtension{Raw: raw}
}

// flatFormatProviderConfig builds a RawExtension in the flat format,
// duplicating each version once per architecture.
//
// Legacy: remove once migration to capability flavors is complete.
func flatFormatProviderConfig(imageName string, archsByVersion map[string][]string) *runtime.RawExtension {
	versions := make([]providerConfigMachineImageVersion, 0)
	for version, archs := range archsByVersion {
		for _, a := range archs {
			versions = append(versions, providerConfigMachineImageVersion{
				Version:      version,
				Architecture: a,
			})
		}
	}
	cfg := providerConfigMachineImages{
		MachineImages: []providerConfigMachineImage{
			{Name: imageName, Versions: versions},
		},
	}
	raw, err := json.Marshal(cfg)
	Expect(err).ToNot(HaveOccurred())
	return &runtime.RawExtension{Raw: raw}
}

// capabilityProviderConfigMultiFlavor builds a capability-flavor RawExtension
// emitting one flavor per architecture per version.
func capabilityProviderConfigMultiFlavor(imageName string, archsByVersion map[string][]string) *runtime.RawExtension {
	versions := make([]providerConfigMachineImageVersion, 0, len(archsByVersion))
	for version, archs := range archsByVersion {
		flavors := make([]providerConfigCapabilityFlavor, 0, len(archs))
		for _, a := range archs {
			flavors = append(flavors, providerConfigCapabilityFlavor{
				Capabilities: providerConfigCapabilities{Architecture: []string{a}},
			})
		}
		versions = append(versions, providerConfigMachineImageVersion{
			Version:           version,
			CapabilityFlavors: flavors,
		})
	}
	cfg := providerConfigMachineImages{
		MachineImages: []providerConfigMachineImage{
			{Name: imageName, Versions: versions},
		},
	}
	raw, err := json.Marshal(cfg)
	Expect(err).ToNot(HaveOccurred())
	return &runtime.RawExtension{Raw: raw}
}

var _ = Describe("machine image version", func() {
	var (
		cloudprofile gardencorev1beta1.CloudProfile
		worker       *gardencorev1beta1.Worker
		expiredTime  metav1.Time
		futureTime   metav1.Time
		rawVersions  []gardencorev1beta1.MachineImageVersion
		currentImage gardencorev1beta1.ShootMachineImage
		arch         string
	)

	Describe("#GetMachineImageVersion", func() {
		BeforeEach(func() {
			expiredTime = metav1.NewTime(time.Now())
			futureTime = metav1.NewTime(time.Now().Add(time.Hour * 24))
			cloudprofile = gardencorev1beta1.CloudProfile{
				Spec: gardencorev1beta1.CloudProfileSpec{
					ProviderConfig: BuildCapabilityProviderConfig(ArchsByImage{imageName: {
						"3.4.5": {arch_amd64, arch_arm64},
						"2.3.3": {arch_amd64, arch_arm64},
						"2.3.4": {arch_amd64, arch_arm64},
						"4.5.6": {arch_amd64, arch_arm64},
					}}),
					MachineImages: []gardencorev1beta1.MachineImage{
						{
							Name: imageName,
							Versions: []gardencorev1beta1.MachineImageVersion{
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "3.4.5",
									ExpirationDate: &futureTime,
								}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "2.3.3",
									ExpirationDate: &futureTime,
								}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "2.3.4",
									ExpirationDate: &futureTime,
								}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "4.5.6",
									ExpirationDate: &expiredTime,
								}},
							},
						},
					},
				},
			}

			worker = &gardencorev1beta1.Worker{
				Machine: gardencorev1beta1.Machine{
					Image: &gardencorev1beta1.ShootMachineImage{
						Name:    imageName,
						Version: ptr.To(common.PatternLatest),
					},
					Architecture: ptr.To(arch_amd64),
				},
			}
		})

		It("should return the latest, not-expired machine image version from a cloudprofile", func() {
			version, err := GetMachineImageVersion(cloudprofile, worker)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("3.4.5"))
		})

		It("should return the latest, not-expired, inplace supported machine image version from a cloudprofile", func() {
			worker.UpdateStrategy = ptr.To(gardencorev1beta1.AutoInPlaceUpdate)
			version, err := GetMachineImageVersion(cloudprofile, worker)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("3.4.5"))
		})

		It("should return the latest-1, not-expired machine image version from a cloudprofile", func() {
			worker.Machine.Image.Version = ptr.To(common.PatternOneMajorBeforeLatest)
			version, err := GetMachineImageVersion(cloudprofile, worker)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("2.3.4"))
		})

		It("should return the version string parsed from the flavor", func() {
			worker.Machine.Image.Version = ptr.To("1.2.3")
			version, err := GetMachineImageVersion(cloudprofile, worker)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("1.2.3"))
		})
	})

	Describe("#getXMajorsBeforeLatestMachineImageVersion", func() {
		BeforeEach(func() {
			rawVersions = []gardencorev1beta1.MachineImageVersion{
				{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: "1.3.4",
					},
				},
				{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: "3.2.4",
					},
				},
				{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: "2.3.4",
					},
				},
				{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: "3.2.3",
					},
				},
				{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: "3.2.4-pre-release",
					},
				},
			}
		})

		It("should return the latest of several machine image versions", func() {
			version, err := getXMajorsBeforeLatestMachineImageVersion(rawVersions, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("3.2.4"))
		})

		It("should return the latest-1 of several machine image versions", func() {
			version, err := getXMajorsBeforeLatestMachineImageVersion(rawVersions, 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("2.3.4"))
		})

		It("should return the latest-2 of several machine image versions", func() {
			version, err := getXMajorsBeforeLatestMachineImageVersion(rawVersions, 2)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("1.3.4"))
		})

		It("should return an error if no matching version is found for latest-x", func() {
			_, err := getXMajorsBeforeLatestMachineImageVersion(rawVersions, 3)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no machine image version matching the pattern latest-3 found"))
		})

		It("should consider full version higher than pre-release and build", func() {
			rawVersions = append(rawVersions,
				gardencorev1beta1.MachineImageVersion{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: "3.2.4+build",
					},
				},
				gardencorev1beta1.MachineImageVersion{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: "3.2.4-pre+build",
					},
				},
			)

			version, err := getXMajorsBeforeLatestMachineImageVersion(rawVersions, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("3.2.4"))

		})

		It("should skip build versions", func() {
			rawVersions = append(rawVersions,
				gardencorev1beta1.MachineImageVersion{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: "3.2.5+build",
					},
				},
				gardencorev1beta1.MachineImageVersion{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: "3.2.5-pre+build",
					},
				},
			)

			version, err := getXMajorsBeforeLatestMachineImageVersion(rawVersions, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("3.2.4"))

		})
	})

	Describe("#GetLatestPreviousVersionForInPlaceUpdate", func() {
		BeforeEach(func() {
			cloudprofile = gardencorev1beta1.CloudProfile{
				Spec: gardencorev1beta1.CloudProfileSpec{
					ProviderConfig: BuildCapabilityProviderConfig(ArchsByImage{imageName: {
						"3.4.5": {arch_amd64},
						"3.4.0": {arch_amd64},
						"2.3.3": {arch_amd64},
						"4.5.6": {arch_amd64},
					}}),
					MachineImages: []gardencorev1beta1.MachineImage{
						{
							Name: imageName,
							Versions: []gardencorev1beta1.MachineImageVersion{
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "3.4.5",
									ExpirationDate: &futureTime,
								}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true, MinVersionForUpdate: ptr.To("2.3.0")}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "3.4.0",
									ExpirationDate: &futureTime,
								}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "2.3.3",
									ExpirationDate: &futureTime,
								}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true, MinVersionForUpdate: ptr.To("2.3.0")}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "4.5.6",
									ExpirationDate: &expiredTime,
								}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true}},
							},
						},
					},
				},
			}

			currentImage = gardencorev1beta1.ShootMachineImage{
				Name:    imageName,
				Version: ptr.To("3.4.5"),
			}
			arch = arch_amd64
		})

		It("should return the latest previous version that supports in-place updates", func() {
			version, err := GetLatestPreviousVersionForInPlaceUpdate(cloudprofile, currentImage, arch)
			Expect(err).ToNot(HaveOccurred())
			Expect(version).To(Equal("3.4.0"))
		})

		It("should return an error if no previous version supports in-place updates", func() {
			currentImage.Version = ptr.To("2.3.3")
			_, err := GetLatestPreviousVersionForInPlaceUpdate(cloudprofile, currentImage, arch)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no machine image versions found that can be in-place updated to the current version"))
		})

		It("should return an error if the current version does not support in-place updates", func() {
			currentImage.Version = ptr.To("4.5.6")
			_, err := GetLatestPreviousVersionForInPlaceUpdate(cloudprofile, currentImage, arch)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("specified machine image version is not found in the cloudprofile or does not support in-place updates"))
		})

		It("should return an error if the current version does not have a minimum version for in-place updates", func() {
			cloudprofile.Spec.MachineImages[0].Versions[0].InPlaceUpdates.MinVersionForUpdate = nil
			_, err := GetLatestPreviousVersionForInPlaceUpdate(cloudprofile, currentImage, arch)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("current machine image version does not have a minimum version for in-place updates"))
		})

		It("should return an error if no machine image versions are found", func() {
			cloudprofile.Spec.MachineImages[0].Versions = []gardencorev1beta1.MachineImageVersion{}
			_, err := GetLatestPreviousVersionForInPlaceUpdate(cloudprofile, currentImage, arch)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no machine image versions found in cloudprofile " + cloudprofile.GetName())) //nolint
		})
	})

	Describe("#FilterMachineImageVersionsByArch", func() {
		versions := []gardencorev1beta1.MachineImageVersion{
			{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: "1.0.0"}},
			{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: "2.0.0"}},
			{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: "3.0.0"}},
		}

		buildCP := func(providerConfig *runtime.RawExtension, capabilityArchs ...string) gardencorev1beta1.CloudProfile {
			cp := gardencorev1beta1.CloudProfile{
				Spec: gardencorev1beta1.CloudProfileSpec{ProviderConfig: providerConfig},
			}
			if len(capabilityArchs) > 0 {
				cp.Spec.MachineCapabilities = []gardencorev1beta1.CapabilityDefinition{
					{Name: "architecture", Values: gardencorev1beta1.CapabilityValues(capabilityArchs)},
				}
			}
			return cp
		}

		DescribeTable("filters versions by architecture from providerConfig",
			func(providerConfig *runtime.RawExtension, expectedAMD64, expectedARM64 []string) {
				cp := buildCP(providerConfig)

				Expect(versionStrings(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_amd64))).
					To(ConsistOf(expectedAMD64))
				Expect(versionStrings(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_arm64))).
					To(ConsistOf(expectedARM64))
			},
			// Legacy formats: remove the next two entries once migration to capability flavors is complete.
			Entry("region format",
				regionFormatProviderConfig(imageName, map[string][]string{
					"1.0.0": {arch_amd64},
					"2.0.0": {arch_arm64},
					"3.0.0": {arch_amd64, arch_arm64},
				}),
				[]string{"1.0.0", "3.0.0"}, []string{"2.0.0", "3.0.0"}),
			Entry("region format defaults missing region.architecture to amd64",
				regionFormatProviderConfig(imageName, map[string][]string{
					"1.0.0": {""}, // single region, no architecture set
					"2.0.0": {arch_arm64},
					"3.0.0": {"", arch_arm64}, // mixed: empty + explicit arm64
				}),
				[]string{"1.0.0", "3.0.0"}, []string{"2.0.0", "3.0.0"}),
			Entry("flat format with duplicate version entries",
				flatFormatProviderConfig(imageName, map[string][]string{
					"1.0.0": {arch_amd64},
					"2.0.0": {arch_amd64, arch_arm64},
					"3.0.0": {arch_arm64},
				}),
				[]string{"1.0.0", "2.0.0"}, []string{"2.0.0", "3.0.0"}),
			Entry("capability-flavor format (single flavor with arch list)",
				BuildCapabilityProviderConfig(ArchsByImage{imageName: {
					"1.0.0": {arch_amd64},
					"2.0.0": {arch_amd64, arch_arm64},
					"3.0.0": {arch_arm64},
				}}),
				[]string{"1.0.0", "2.0.0"}, []string{"2.0.0", "3.0.0"}),
			Entry("capability-flavor format (multiple flavors per version)",
				capabilityProviderConfigMultiFlavor(imageName, map[string][]string{
					"1.0.0": {arch_amd64},
					"2.0.0": {arch_amd64, arch_arm64},
					"3.0.0": {arch_arm64},
				}),
				[]string{"1.0.0", "2.0.0"}, []string{"2.0.0", "3.0.0"}),
		)

		It("returns all versions for the implicit arch when MachineCapabilities[architecture] has a single value", func() {
			cp := buildCP(nil, arch_amd64)

			Expect(versionStrings(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_amd64))).
				To(ConsistOf("1.0.0", "2.0.0", "3.0.0"))
			Expect(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_arm64)).To(BeEmpty())
		})

		It("ignores providerConfig when MachineCapabilities[architecture] has a single value", func() {
			// providerConfig declares arm64 on every version, but MachineCapabilities pins amd64.
			cp := buildCP(
				BuildCapabilityProviderConfig(ArchsByImage{imageName: {
					"1.0.0": {arch_arm64},
					"2.0.0": {arch_arm64},
					"3.0.0": {arch_arm64},
				}}),
				arch_amd64,
			)

			Expect(versionStrings(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_amd64))).
				To(ConsistOf("1.0.0", "2.0.0", "3.0.0"))
			Expect(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_arm64)).To(BeEmpty())
		})

		It("falls back to providerConfig when MachineCapabilities[architecture] has more than one value", func() {
			cp := buildCP(
				BuildCapabilityProviderConfig(ArchsByImage{imageName: {
					"1.0.0": {arch_amd64},
					"2.0.0": {arch_arm64},
					"3.0.0": {arch_amd64, arch_arm64},
				}}),
				arch_amd64, arch_arm64,
			)

			Expect(versionStrings(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_amd64))).
				To(ConsistOf("1.0.0", "3.0.0"))
			Expect(versionStrings(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_arm64))).
				To(ConsistOf("2.0.0", "3.0.0"))
		})

		It("returns an empty result when the cloudprofile has no provider config", func() {
			cp := buildCP(nil)
			Expect(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_amd64)).To(BeEmpty())
		})

		It("ignores provider-config entries for other image names", func() {
			cp := buildCP(BuildCapabilityProviderConfig(ArchsByImage{"other-image": {
				"1.0.0": {arch_amd64},
			}}))
			Expect(FilterMachineImageVersionsByArch(cp, imageName, versions, arch_amd64)).To(BeEmpty())
		})
	})
})

func versionStrings(versions []gardencorev1beta1.MachineImageVersion) []string {
	result := make([]string, 0, len(versions))
	for _, v := range versions {
		result = append(result, v.Version)
	}
	return result
}
