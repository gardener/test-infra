package util

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("machine image version test", func() {

	var (
		rawVersions []gardencorev1beta1.MachineImageVersion
	)

	BeforeEach(func() {
		rawVersions = []gardencorev1beta1.MachineImageVersion{
			gardencorev1beta1.MachineImageVersion{
				ExpirableVersion: gardencorev1beta1.ExpirableVersion{
					Version: "1.2.3",
				},
			},
			gardencorev1beta1.MachineImageVersion{
				ExpirableVersion: gardencorev1beta1.ExpirableVersion{
					Version: "1.2.4",
				},
			},
			gardencorev1beta1.MachineImageVersion{
				ExpirableVersion: gardencorev1beta1.ExpirableVersion{
					Version: "1.2.4-pre-release",
				},
			},
		}
	})

	It("should return the latest of two machine image versions", func() {
		version, err := getLatestMachineImageVersion(rawVersions)
		Expect(err).ToNot(HaveOccurred())
		Expect(version.Version).To(Equal("1.2.4"))
	})

	It("should consider full version higher than pre-release and build", func() {
		rawVersions = append(rawVersions,
			gardencorev1beta1.MachineImageVersion{
				ExpirableVersion: gardencorev1beta1.ExpirableVersion{
					Version: "1.2.4+build",
				},
			},
			gardencorev1beta1.MachineImageVersion{
				ExpirableVersion: gardencorev1beta1.ExpirableVersion{
					Version: "1.2.4-pre+build",
				},
			},
		)

		version, err := getLatestMachineImageVersion(rawVersions)
		Expect(err).ToNot(HaveOccurred())
		Expect(version.Version).To(Equal("1.2.4"))

	})

	It("should skip build versions", func() {
		rawVersions = append(rawVersions,
			gardencorev1beta1.MachineImageVersion{
				ExpirableVersion: gardencorev1beta1.ExpirableVersion{
					Version: "1.2.5+build",
				},
			},
			gardencorev1beta1.MachineImageVersion{
				ExpirableVersion: gardencorev1beta1.ExpirableVersion{
					Version: "1.2.5-pre+build",
				},
			},
		)

		version, err := getLatestMachineImageVersion(rawVersions)
		Expect(err).ToNot(HaveOccurred())
		Expect(version.Version).To(Equal("1.2.4"))

	})
})
