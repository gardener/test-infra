// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package result

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v83/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	trerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/logger"
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

// newTestGithubClient creates a github.Client pointing at the given httptest.Server.
func newTestGithubClient(serverURL string) *github.Client {
	client, err := github.NewClient(nil).WithEnterpriseURLs(serverURL+"/", serverURL+"/")
	Expect(err).ToNot(HaveOccurred())
	return client
}

var _ = Describe("getRelease retry", func() {
	var (
		origSleepFunc       func(time.Duration)
		origMaxDuration     time.Duration
		origInitialInterval time.Duration
	)

	BeforeEach(func() {
		// Save original values
		origSleepFunc = retrySleepFunc
		origMaxDuration = retryMaxDuration
		origInitialInterval = retryInitialInterval

		// Override sleep to be a no-op so tests run fast
		retrySleepFunc = func(d time.Duration) {}
		// Use short durations for testing
		retryMaxDuration = 500 * time.Millisecond
		retryInitialInterval = 10 * time.Millisecond

		// Initialize logger for tests
		log, err := logger.New(&logger.Config{Development: true})
		Expect(err).ToNot(HaveOccurred())
		logger.SetLogger(log)
	})

	AfterEach(func() {
		// Restore original values
		retrySleepFunc = origSleepFunc
		retryMaxDuration = origMaxDuration
		retryInitialInterval = origInitialInterval
	})

	It("should return release on first attempt when GetReleaseByTag succeeds", func() {
		releaseID := int64(12345)
		tagName := "1.0.0"
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v3/repos/owner/repo/releases/tags/"+tagName, func(w http.ResponseWriter, r *http.Request) {
			release := &github.RepositoryRelease{
				ID:      &releaseID,
				TagName: &tagName,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(release)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestGithubClient(server.URL)
		release, err := getRelease(client, "owner", "repo", "1.0.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(release).ToNot(BeNil())
		Expect(*release.ID).To(Equal(releaseID))
	})

	It("should retry and succeed when GetReleaseByTag fails initially then succeeds", func() {
		releaseID := int64(67890)
		tagName := "2.0.0"
		var callCount int32

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v3/repos/owner/repo/releases/tags/"+tagName, func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&callCount, 1)
			if count <= 2 {
				// Fail the first 2 attempts
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprint(w, `{"message":"Not Found"}`)
				return
			}
			// Succeed on the 3rd attempt
			release := &github.RepositoryRelease{
				ID:      &releaseID,
				TagName: &tagName,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(release)
		})
		// ListReleases also needs to return empty so the fallback search doesn't find anything
		mux.HandleFunc("/api/v3/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		// Allow enough time for retries
		retryMaxDuration = 5 * time.Second

		client := newTestGithubClient(server.URL)
		release, err := getRelease(client, "owner", "repo", "2.0.0")
		Expect(err).ToNot(HaveOccurred())
		Expect(release).ToNot(BeNil())
		Expect(*release.ID).To(Equal(releaseID))
		Expect(atomic.LoadInt32(&callCount)).To(Equal(int32(3)))
	})

	It("should give up after max retry duration when release is never found", func() {
		tagName := "3.0.0"
		var callCount int32

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v3/repos/owner/repo/releases/tags/"+tagName, func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&callCount, 1)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"Not Found"}`)
		})
		mux.HandleFunc("/api/v3/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		// Very short max duration to fail quickly
		retryMaxDuration = 50 * time.Millisecond
		retryInitialInterval = 10 * time.Millisecond

		client := newTestGithubClient(server.URL)
		release, err := getRelease(client, "owner", "repo", "3.0.0")
		Expect(err).To(HaveOccurred())
		Expect(release).To(BeNil())
		Expect(err.Error()).To(ContainSubstring("failed to find release after retrying"))
		// Should have attempted at least once
		Expect(atomic.LoadInt32(&callCount)).To(BeNumerically(">=", 1))
	})

	It("should find a draft release by name when GetReleaseByTag is not used for prerelease versions", func() {
		releaseID := int64(11111)
		releaseName := "4.0.0"
		tagName := "4.0.0-dev-abc123"
		isDraft := true

		mux := http.NewServeMux()
		mux.HandleFunc("/api/v3/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
			releases := []*github.RepositoryRelease{
				{
					ID:      &releaseID,
					Name:    &releaseName,
					TagName: &tagName,
					Draft:   &isDraft,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(releases)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestGithubClient(server.URL)
		// Using a prerelease version so it skips GetReleaseByTag and goes to ListReleases
		release, err := getRelease(client, "owner", "repo", "4.0.0-dev-abc123")
		Expect(err).ToNot(HaveOccurred())
		Expect(release).ToNot(BeNil())
		Expect(*release.ID).To(Equal(releaseID))
	})

	It("should increment sleep duration on each retry", func() {
		var sleepDurations []time.Duration
		retrySleepFunc = func(d time.Duration) {
			sleepDurations = append(sleepDurations, d)
		}
		retryMaxDuration = 200 * time.Millisecond
		retryInitialInterval = 10 * time.Millisecond

		tagName := "5.0.0"
		mux := http.NewServeMux()
		mux.HandleFunc("/api/v3/repos/owner/repo/releases/tags/"+tagName, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"message":"Not Found"}`)
		})
		mux.HandleFunc("/api/v3/repos/owner/repo/releases", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `[]`)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		client := newTestGithubClient(server.URL)
		_, _ = getRelease(client, "owner", "repo", "5.0.0")

		// Verify sleep durations are incrementing
		Expect(len(sleepDurations)).To(BeNumerically(">=", 1))
		for i := 1; i < len(sleepDurations); i++ {
			Expect(sleepDurations[i]).To(BeNumerically(">", sleepDurations[i-1]))
		}
	})
})
