package testrunner_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/testrunner"
)

var _ = Describe("Testrunner Utility Functions", func() {

	Context("Read CloudProfiles from Disk", func() {

		It("should read all cloudprofiles from the testdata directory", func() {
			cloudProfiles, err := testrunner.GetCloudProfilesFromDisk("./testdata/util/cloudprofiles")
			Expect(err).ToNot(HaveOccurred())
			Expect(cloudProfiles).To(HaveLen(2))
			Expect(cloudProfiles).To(HaveKey("cloudprofile1"))
			Expect(cloudProfiles).To(HaveKey("cloudprofile2"))

			// Validate cloudprofile1 properties
			cp1 := cloudProfiles["cloudprofile1"]
			Expect(cp1.Name).To(Equal("cloudprofile1"))
			Expect(cp1.Spec.Type).To(Equal("test"))
			Expect(cp1.Spec.Kubernetes.Versions).To(HaveLen(4))
			Expect(cp1.Spec.Kubernetes.Versions[0].Version).To(Equal("1.33.2"))
			Expect(cp1.Spec.MachineImages).To(HaveLen(1))
			Expect(cp1.Spec.MachineImages[0].Name).To(Equal("gardenlinux"))
			Expect(cp1.Spec.MachineTypes).To(HaveLen(1))
			Expect(cp1.Spec.MachineTypes[0].Name).To(Equal("medium"))
			Expect(cp1.Spec.Regions).To(HaveLen(1))
			Expect(cp1.Spec.Regions[0].Name).To(Equal("europe-central-1"))

			// Validate cloudprofile2 properties
			cp2 := cloudProfiles["cloudprofile2"]
			Expect(cp2.Name).To(Equal("cloudprofile2"))
			Expect(cp2.Spec.Type).To(Equal("test2"))
			Expect(cp2.Spec.Kubernetes.Versions).To(HaveLen(4))
			Expect(cp2.Spec.MachineImages).To(HaveLen(1))
			Expect(cp2.Spec.MachineTypes).To(HaveLen(1))
			Expect(cp2.Spec.Regions).To(HaveLen(1))
		})

		It("should return an error for non-existent directory", func() {
			cloudProfiles, err := testrunner.GetCloudProfilesFromDisk("./testdata/util/nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(cloudProfiles).To(BeEmpty())
		})

		It("should return an empty list no CloudProfiles are found in the subtree", func() {
			cloudProfiles, err := testrunner.GetCloudProfilesFromDisk("./testdata/util/cloudprofiles/no_cloud_profiles")
			Expect(err).ToNot(HaveOccurred())
			Expect(cloudProfiles).To(BeEmpty())
		})
	})

})
