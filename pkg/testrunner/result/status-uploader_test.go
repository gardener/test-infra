// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package result

import (
	"encoding/json"
	"os"
	"path/filepath"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	trerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
)

func newRun(cloudProvider string, phase argov1.WorkflowPhase, err error) *testrunner.Run {
	return &testrunner.Run{
		Testrun: &tmv1beta1.Testrun{
			Status: tmv1beta1.TestrunStatus{
				Phase: phase,
			},
		},
		Metadata: &metadata.Metadata{
			Landscape:     "dev",
			CloudProvider: cloudProvider,
		},
		Error: err,
	}
}

var _ = Describe("StoreResultsAsFiles", func() {

	var (
		destDir     string
		assetPrefix string
	)

	BeforeEach(func() {
		var err error
		destDir, err = os.MkdirTemp("", "store-results-test-*")
		Expect(err).ToNot(HaveOccurred())
		assetPrefix = "prefix-"
	})

	AfterEach(func() {
		Expect(os.RemoveAll(destDir)).To(Succeed())
	})

	It("should create overview and status files for a single successful run", func() {
		runs := testrunner.RunList{
			newRun("aws", tmv1beta1.RunPhaseSuccess, nil),
		}

		Expect(StoreResultsAsFiles(logr.Discard(), runs, assetPrefix, destDir)).To(Succeed())

		overviewPath := filepath.Join(destDir, assetPrefix+"dev_overview.json")
		Expect(overviewPath).To(BeAnExistingFile())

		overviewFile, err := os.ReadFile(overviewPath)
		Expect(err).ToNot(HaveOccurred())
		var overview AssetOverview
		Expect(json.Unmarshal(overviewFile, &overview)).To(Succeed())
		Expect(overview.AssetOverviewItems).To(HaveLen(1))
		Expect(overview.AssetOverviewItems[0].Successful).To(BeTrue())

		assetName := generateTestrunAssetName(*runs[0], assetPrefix)
		statusFile := filepath.Join(destDir, assetPrefix+"dev", assetName)
		Expect(statusFile).To(BeAnExistingFile())
	})

	It("should create separate status files for multiple runs", func() {
		runs := testrunner.RunList{
			newRun("aws", tmv1beta1.RunPhaseSuccess, nil),
			newRun("gcp", tmv1beta1.RunPhaseFailed, nil),
		}

		Expect(StoreResultsAsFiles(logr.Discard(), runs, assetPrefix, destDir)).To(Succeed())

		overviewPath := filepath.Join(destDir, assetPrefix+"dev_overview.json")
		Expect(overviewPath).To(BeAnExistingFile())

		overviewFile, err := os.ReadFile(overviewPath)
		Expect(err).ToNot(HaveOccurred())
		var overview AssetOverview
		Expect(json.Unmarshal(overviewFile, &overview)).To(Succeed())
		Expect(overview.AssetOverviewItems).To(HaveLen(2))

		for _, run := range runs {
			assetName := generateTestrunAssetName(*run, assetPrefix)
			statusFile := filepath.Join(destDir, assetPrefix+"dev", assetName)
			Expect(statusFile).To(BeAnExistingFile())
		}
	})

	It("should return nil and create no files when all runs have non-timeout errors", func() {
		runs := testrunner.RunList{
			newRun("aws", tmv1beta1.RunPhaseFailed, trerrors.NewNotFoundError("not found")),
		}

		Expect(StoreResultsAsFiles(logr.Discard(), runs, assetPrefix, destDir)).To(Succeed())

		overviewPath := filepath.Join(destDir, assetPrefix+"dev_overview.json")
		Expect(overviewPath).ToNot(BeAnExistingFile())
	})

	It("should include runs with timeout errors", func() {
		runs := testrunner.RunList{
			newRun("aws", tmv1beta1.RunPhaseFailed, trerrors.NewTimeoutError("timed out")),
		}

		Expect(StoreResultsAsFiles(logr.Discard(), runs, assetPrefix, destDir)).To(Succeed())

		overviewPath := filepath.Join(destDir, assetPrefix+"dev_overview.json")
		Expect(overviewPath).To(BeAnExistingFile())
	})

	It("should extend an existing overview file with new runs", func() {
		run := newRun("aws", tmv1beta1.RunPhaseSuccess, nil)
		runs := testrunner.RunList{run}

		// First call creates the overview
		Expect(StoreResultsAsFiles(logr.Discard(), runs, assetPrefix, destDir)).To(Succeed())

		// Second call with a different run appends to the existing overview
		run2 := newRun("gcp", tmv1beta1.RunPhaseSuccess, nil)
		runs2 := testrunner.RunList{run2}
		Expect(StoreResultsAsFiles(logr.Discard(), runs2, assetPrefix, destDir)).To(Succeed())

		overviewPath := filepath.Join(destDir, assetPrefix+"dev_overview.json")
		overviewFile, err := os.ReadFile(overviewPath)
		Expect(err).ToNot(HaveOccurred())
		var overview AssetOverview
		Expect(json.Unmarshal(overviewFile, &overview)).To(Succeed())
		Expect(overview.AssetOverviewItems).To(HaveLen(2))
	})

	It("should mark failed run as not successful in overview", func() {
		runs := testrunner.RunList{
			newRun("aws", tmv1beta1.RunPhaseFailed, nil),
		}

		Expect(StoreResultsAsFiles(logr.Discard(), runs, assetPrefix, destDir)).To(Succeed())

		overviewPath := filepath.Join(destDir, assetPrefix+"dev_overview.json")
		overviewFile, err := os.ReadFile(overviewPath)
		Expect(err).ToNot(HaveOccurred())
		var overview AssetOverview
		Expect(json.Unmarshal(overviewFile, &overview)).To(Succeed())
		Expect(overview.AssetOverviewItems).To(HaveLen(1))
		Expect(overview.AssetOverviewItems[0].Successful).To(BeFalse())
	})

	It("should clean and recreate the archive content directory on repeated calls", func() {
		runs := testrunner.RunList{
			newRun("aws", tmv1beta1.RunPhaseSuccess, nil),
		}

		Expect(StoreResultsAsFiles(logr.Discard(), runs, assetPrefix, destDir)).To(Succeed())

		archiveContentDir := filepath.Join(destDir, assetPrefix+"dev")
		staleFile := filepath.Join(archiveContentDir, "stale.txt")
		Expect(os.WriteFile(staleFile, []byte("stale"), 0600)).To(Succeed())

		Expect(StoreResultsAsFiles(logr.Discard(), runs, assetPrefix, destDir)).To(Succeed())

		Expect(staleFile).ToNot(BeAnExistingFile())
	})

	It("should return an error when resultsDirectoryPath does not exist", func() {
		runs := testrunner.RunList{
			newRun("aws", tmv1beta1.RunPhaseSuccess, nil),
		}

		nonExistentParent := filepath.Join(destDir, "nonexistent", "subdir")
		err := StoreResultsAsFiles(logr.Discard(), runs, assetPrefix, nonExistentParent)
		Expect(err).To(HaveOccurred())
	})
})
