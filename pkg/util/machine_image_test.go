package util

import (
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
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

var _ = Describe("get machine images from a cloudprofile", func() {
	var (
		cloudprofile gardencorev1beta1.CloudProfile
		worker       *gardencorev1beta1.Worker
		expiredTime  metav1.Time
		futureTime   metav1.Time
	)

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

})

var _ = Describe("machine image version test", func() {

	var (
		rawVersions []gardencorev1beta1.MachineImageVersion
	)

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
