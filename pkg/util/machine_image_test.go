package util

import (
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"github.com/gardener/test-infra/pkg/common"
)

const (
	imageName  = "gardenlinux"
	arch_amd64 = "amd64"
	arch_arm64 = "arm64"
)

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
					MachineImages: []gardencorev1beta1.MachineImage{
						{
							Name: imageName,
							Versions: []gardencorev1beta1.MachineImageVersion{
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "3.4.5",
									ExpirationDate: &futureTime,
								}, Architectures: []string{arch_amd64, arch_arm64}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "2.3.3",
									ExpirationDate: &futureTime,
								}, Architectures: []string{arch_amd64, arch_arm64}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "2.3.4",
									ExpirationDate: &futureTime,
								}, Architectures: []string{arch_amd64, arch_arm64}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "4.5.6",
									ExpirationDate: &expiredTime,
								}, Architectures: []string{arch_amd64, arch_arm64}},
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

		It("should resolve latest version when architectures are declared via machineCapabilities (no per-version architectures field)", func() {
			// Simulate the AWS CloudProfile format where architectures are declared globally
			// via spec.machineCapabilities instead of per machine image version.
			cloudprofile.Spec.MachineCapabilities = []gardencorev1beta1.CapabilityDefinition{
				{
					Name:   v1beta1constants.ArchitectureName,
					Values: gardencorev1beta1.CapabilityValues{arch_amd64, arch_arm64},
				},
			}
			for i := range cloudprofile.Spec.MachineImages[0].Versions {
				cloudprofile.Spec.MachineImages[0].Versions[i].Architectures = nil
			}
			version, err := GetMachineImageVersion(cloudprofile, worker)
			Expect(err).ToNot(HaveOccurred())
			Expect(version.Version).To(Equal("3.4.5"))
		})

		It("should fail to resolve when machineCapabilities does not list the requested architecture", func() {
			cloudprofile.Spec.MachineCapabilities = []gardencorev1beta1.CapabilityDefinition{
				{
					Name:   v1beta1constants.ArchitectureName,
					Values: gardencorev1beta1.CapabilityValues{arch_amd64},
				},
			}
			for i := range cloudprofile.Spec.MachineImages[0].Versions {
				cloudprofile.Spec.MachineImages[0].Versions[i].Architectures = nil
			}
			worker.Machine.Architecture = ptr.To(arch_arm64)
			_, err := GetMachineImageVersion(cloudprofile, worker)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no machine image versions found"))
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

	Describe("#FilterArchSpecificMachineImage", func() {
		var versions []gardencorev1beta1.MachineImageVersion

		BeforeEach(func() {
			versions = []gardencorev1beta1.MachineImageVersion{
				{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: "1.0.0"}, Architectures: []string{arch_amd64}},
				{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: "2.0.0"}, Architectures: []string{arch_amd64, arch_arm64}},
				{ExpirableVersion: gardencorev1beta1.ExpirableVersion{Version: "3.0.0"}, Architectures: []string{arch_arm64}},
			}
		})

		It("should filter versions by per-version architectures when machineCapabilities is nil", func() {
			result := FilterArchSpecificMachineImage(versions, arch_amd64, nil)
			Expect(result).To(HaveLen(2))
			Expect(result[0].Version).To(Equal("1.0.0"))
			Expect(result[1].Version).To(Equal("2.0.0"))
		})

		It("should return all versions when machineCapabilities declares the requested architecture", func() {
			caps := []gardencorev1beta1.CapabilityDefinition{
				{
					Name:   v1beta1constants.ArchitectureName,
					Values: gardencorev1beta1.CapabilityValues{arch_amd64, arch_arm64},
				},
			}
			// Versions have no per-version architectures (AWS-style)
			for i := range versions {
				versions[i].Architectures = nil
			}
			result := FilterArchSpecificMachineImage(versions, arch_amd64, caps)
			Expect(result).To(HaveLen(3))
		})

		It("should return a new slice when machineCapabilities match, not the original", func() {
			caps := []gardencorev1beta1.CapabilityDefinition{
				{
					Name:   v1beta1constants.ArchitectureName,
					Values: gardencorev1beta1.CapabilityValues{arch_amd64},
				},
			}
			for i := range versions {
				versions[i].Architectures = nil
			}
			result := FilterArchSpecificMachineImage(versions, arch_amd64, caps)
			Expect(&result[0]).ToNot(BeIdenticalTo(&versions[0]))
		})

		It("should fall through to per-version filter when machineCapabilities does not list the requested architecture", func() {
			caps := []gardencorev1beta1.CapabilityDefinition{
				{
					Name:   v1beta1constants.ArchitectureName,
					Values: gardencorev1beta1.CapabilityValues{arch_amd64},
				},
			}
			result := FilterArchSpecificMachineImage(versions, arch_arm64, caps)
			Expect(result).To(HaveLen(2))
			Expect(result[0].Version).To(Equal("2.0.0"))
			Expect(result[1].Version).To(Equal("3.0.0"))
		})

		It("should return empty when no versions match the architecture and machineCapabilities is nil", func() {
			for i := range versions {
				versions[i].Architectures = nil
			}
			result := FilterArchSpecificMachineImage(versions, arch_amd64, nil)
			Expect(result).To(BeEmpty())
		})
	})

	Describe("#GetLatestPreviousVersionForInPlaceUpdate", func() {
		BeforeEach(func() {
			cloudprofile = gardencorev1beta1.CloudProfile{
				Spec: gardencorev1beta1.CloudProfileSpec{
					MachineImages: []gardencorev1beta1.MachineImage{
						{
							Name: imageName,
							Versions: []gardencorev1beta1.MachineImageVersion{
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "3.4.5",
									ExpirationDate: &futureTime,
								}, Architectures: []string{arch_amd64}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true, MinVersionForUpdate: ptr.To("2.3.0")}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "3.4.0",
									ExpirationDate: &futureTime,
								}, Architectures: []string{arch_amd64}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "2.3.3",
									ExpirationDate: &futureTime,
								}, Architectures: []string{arch_amd64}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true, MinVersionForUpdate: ptr.To("2.3.0")}},
								{ExpirableVersion: gardencorev1beta1.ExpirableVersion{
									Version:        "4.5.6",
									ExpirationDate: &expiredTime,
								}, Architectures: []string{arch_amd64}, InPlaceUpdates: &gardencorev1beta1.InPlaceUpdates{Supported: true}},
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
			Expect(err.Error()).To(Equal("no machine image versions found in cloudprofile " + cloudprofile.GetName()))
		})
	})
})
