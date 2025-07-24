package testrun_renderer

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
)

const (
	locationSetName = "default"
)

func TestLocationRenderer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AddLocationsToTestrun Test Suite")
}

var _ = Describe("AddLocationsToTestrun Test", func() {

	var (
		baseComponentDescriptor []*componentdescriptor.Component
		baseTestrun             *tmv1beta1.Testrun
		additionalLocations     []common.AdditionalLocation
	)

	BeforeEach(func() {
		baseComponentDescriptor = []*componentdescriptor.Component{
			{
				Name:    "example.com/repo1",
				Version: "v1.2.3",
			},
			{
				Name:    "example.com/repo2",
				Version: "v1.2.3",
			},
			{
				Name:    "example.com/repo3",
				Version: "v1.2.3",
			},
			{
				Name:           "example.com/repo4",
				Version:        "v1.2.3",
				SourceRevision: "v1.2.3",
				SourceRepoURL:  "example.com/repo4",
			},
		}

		baseTestrun = &tmv1beta1.Testrun{
			Spec: tmv1beta1.TestrunSpec{},
		}

		additionalLocations = []common.AdditionalLocation{}

	})

	It("Should parse all components into a locationSet", func() {

		err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(baseTestrun.Spec.LocationSets)).To(Equal(1))
		Expect(len(baseTestrun.Spec.LocationSets[0].Locations)).To(Equal(4))
	})

	It("Should only have duplicates when additionalLocations are specified", func() {
		additionalLocations = append(additionalLocations, common.AdditionalLocation{
			Type:     "git",
			Repo:     "https://example/com/add1",
			Revision: "v1.2.3",
		})
		err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
		Expect(err).ToNot(HaveOccurred())
		Expect(len(baseTestrun.Spec.LocationSets)).To(Equal(1))
		Expect(len(baseTestrun.Spec.LocationSets[0].Locations)).To(Equal(5))
	})

	It("Should use the source information", func() {
		sourceRepo := "example.com/different/repo5"
		sourceVersion := "v1.2.3"
		baseComponentDescriptor = append(baseComponentDescriptor, &componentdescriptor.Component{
			Name:           "example.com/repo5",
			Version:        "v1.2.2",
			SourceRevision: sourceVersion,
			SourceRepoURL:  sourceRepo,
		})

		err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
		Expect(err).ToNot(HaveOccurred())
		for _, location := range baseTestrun.Spec.LocationSets[0].Locations {
			if location.Repo == sourceRepo {
				Expect(location.Revision).To(Equal(sourceVersion))
			}
		}
	})

	Describe("A repository identified by a component name appears with two different version in the component descriptor", func() {
		It("Should pick the initial version, if it is higher", func() {
			repo := "example.com/repo1"
			baseComponentDescriptor = append(baseComponentDescriptor, &componentdescriptor.Component{
				Name:    repo,
				Version: "v1.1.2",
			})
			err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(baseTestrun.Spec.LocationSets)).To(Equal(1))
			Expect(len(baseTestrun.Spec.LocationSets[0].Locations)).To(Equal(4))
			for _, location := range baseTestrun.Spec.LocationSets[0].Locations {
				if location.Repo == repo {
					Expect(location.Revision).To(Equal("v1.2.3"))
				}
			}
		})

		It("Should pick the incoming version, if it is higher", func() {
			repo := "example.com/repo1"
			version := "v1.3.3"
			baseComponentDescriptor = append(baseComponentDescriptor, &componentdescriptor.Component{
				Name:    repo,
				Version: version,
			})
			err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(baseTestrun.Spec.LocationSets)).To(Equal(1))
			Expect(len(baseTestrun.Spec.LocationSets[0].Locations)).To(Equal(4))
			for _, location := range baseTestrun.Spec.LocationSets[0].Locations {
				if location.Repo == repo {
					Expect(location.Revision).To(Equal(version))
				}
			}
		})

		It("Should pick the master branch", func() {
			repo1 := "example.com/repo1"
			repo2 := "example.com/repo2"
			version := "master"
			baseComponentDescriptor[1].Version = version
			baseComponentDescriptor = append(baseComponentDescriptor,
				&componentdescriptor.Component{
					Name:    repo1,
					Version: version,
				},
				&componentdescriptor.Component{
					Name:    repo2,
					Version: "v1.2.3",
				})
			err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(baseTestrun.Spec.LocationSets)).To(Equal(1))
			Expect(len(baseTestrun.Spec.LocationSets[0].Locations)).To(Equal(4))
			for _, location := range baseTestrun.Spec.LocationSets[0].Locations {
				if location.Repo == repo1 {
					Expect(location.Revision).To(Equal(version))
				}
				if location.Repo == repo2 {
					Expect(location.Revision).To(Equal(version))
				}
			}
		})

		It("Should pick the main branch", func() {
			repo1 := "example.com/repo1"
			repo2 := "example.com/repo2"
			version := "main"
			baseComponentDescriptor[1].Version = version
			baseComponentDescriptor = append(baseComponentDescriptor,
				&componentdescriptor.Component{
					Name:    repo1,
					Version: version,
				},
				&componentdescriptor.Component{
					Name:    repo2,
					Version: "v1.2.3",
				})
			err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(baseTestrun.Spec.LocationSets)).To(Equal(1))
			Expect(len(baseTestrun.Spec.LocationSets[0].Locations)).To(Equal(4))
			for _, location := range baseTestrun.Spec.LocationSets[0].Locations {
				if location.Repo == repo1 {
					Expect(location.Revision).To(Equal(version))
				}
				if location.Repo == repo2 {
					Expect(location.Revision).To(Equal(version))
				}
			}
		})
		It("Should keep the initial version when semVer parsing fails", func() {
			repo1 := "example.com/repo1"
			repo4 := "example.com/repo4"
			version := "abc"
			baseComponentDescriptor = append(baseComponentDescriptor,
				&componentdescriptor.Component{
					Name:    repo1,
					Version: version,
				},
				&componentdescriptor.Component{
					Name:    repo4,
					Version: version,
				},
				&componentdescriptor.Component{
					Name:    repo4,
					Version: "v1.2.3",
				},
			)
			err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(baseTestrun.Spec.LocationSets)).To(Equal(1))
			Expect(len(baseTestrun.Spec.LocationSets[0].Locations)).To(Equal(4))
			for _, location := range baseTestrun.Spec.LocationSets[0].Locations {
				if location.Repo == repo1 {
					Expect(location.Revision).To(Equal("v1.2.3"))
				}
				if location.Repo == repo4 {
					Expect(location.Revision).To(Equal(version))
				}
			}
		})

		Describe("A repository identified by a source appears with two different version in the component descriptor", func() {
			It("Should pick the initial version, if it is higher", func() {
				repo := "example.com/repo4"
				baseComponentDescriptor = append(baseComponentDescriptor, &componentdescriptor.Component{
					Name:           repo,
					Version:        "v1.1.2",
					SourceRevision: "v1.2.2",
					SourceRepoURL:  repo,
				})
				err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(baseTestrun.Spec.LocationSets)).To(Equal(1))
				Expect(len(baseTestrun.Spec.LocationSets[0].Locations)).To(Equal(4))
				for _, location := range baseTestrun.Spec.LocationSets[0].Locations {
					if location.Repo == repo {
						Expect(location.Revision).To(Equal("v1.2.3"))
					}
				}
			})

			It("Should pick the incoming version, if it is higher", func() {
				repo := "example.com/repo4"
				version := "v1.3.3"
				sourceVersion := version
				sourceRepoURL := repo
				baseComponentDescriptor = append(baseComponentDescriptor, &componentdescriptor.Component{
					Name:           repo,
					Version:        version,
					SourceRepoURL:  sourceRepoURL,
					SourceRevision: sourceVersion,
				})
				err := AddLocationsToTestrun(baseTestrun, locationSetName, baseComponentDescriptor, true, additionalLocations)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(baseTestrun.Spec.LocationSets)).To(Equal(1))
				Expect(len(baseTestrun.Spec.LocationSets[0].Locations)).To(Equal(4))
				for _, location := range baseTestrun.Spec.LocationSets[0].Locations {
					if location.Repo == sourceRepoURL {
						Expect(location.Revision).To(Equal(sourceVersion))
					}
				}
			})
		})
	})
})
