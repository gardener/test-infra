// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	tiutil "github.com/gardener/test-infra/pkg/util"
)

var (
	ConformanceLogLevel      int
	HydrophoneVersion        string
	K8sRelease               string
	SkipIndividualTestCases  string
	GinkgoParallel           bool
	FlakeAttempts            int
	ExportPath               string
	PublishResultsToTestgrid bool
	ShootKubeconfigPath      string
	CloudProvider            string
	GardenerVersion          string
	DryRun                   bool
	GcsBucket                string
	GcsProjectID             string
)

func init() {
	flag.IntVar(&ConformanceLogLevel, "conformanceLogLevel", 2, "Hydrophone log level for conformance test")
	flag.StringVar(&HydrophoneVersion, "hydrophoneVersion", "", "Hydrophone version to be used")
	flag.StringVar(&K8sRelease, "k8sVersion", "", "Kubernetes release version e.g. 1.14.0")
	flag.StringVar(&SkipIndividualTestCases, "skipIndividualTestCases", "", "A list of ginkgo.skip patterns (regex based) to skip individual test cases, use \"|\" as delimiter.")
	flag.IntVar(&FlakeAttempts, "flakeAttempts", 1, "Testcase flake attempts. Will run testcase n times, until it is successful")
	flag.StringVar(&ShootKubeconfigPath, "kubeconfig", "", "Kubeconfig file path of cluster to test")
	flag.StringVar(&CloudProvider, "cloudprovider", "", "Cluster cloud provider (aws, gcp, azure, alicloud, openstack)")
	flag.BoolVar(&DryRun, "dryRun", false, "use in combination with --conformanceLogLevel to output testcases")
	flag.Parse()

	if K8sRelease == "" {
		K8sRelease = os.Getenv("K8S_VERSION")
	}
	if K8sRelease == "" {
		log.Fatal("K8S_VERSION environment variable not found")
	}

	if HydrophoneVersion == "" {
		HydrophoneVersion = os.Getenv("HYDROPHONE_VERSION")
	}

	if HydrophoneVersion == "" {
		HydrophoneVersion = "latest"
	}

	if SkipIndividualTestCases == "" {
		SkipIndividualTestCases = os.Getenv("SKIP_INDIVIDUAL_TEST_CASES")
	}

	GinkgoParallel = tiutil.GetenvBool("GINKGO_PARALLEL", true)

	PublishResultsToTestgrid = tiutil.GetenvBool("PUBLISH_RESULTS_TO_TESTGRID", false)
	GcsBucket = tiutil.Getenv("GCS_BUCKET", "k8s-conformance-gardener")
	GcsProjectID = tiutil.Getenv("GCS_PROJECT_ID", "gardener")

	if ShootKubeconfigPath == "" {
		ShootKubeconfigPath = tiutil.Getenv("E2E_KUBECONFIG_PATH", os.Getenv("KUBECONFIG"))
	}
	if ShootKubeconfigPath == "" {
		log.Fatal("shoot config not set")
	}
	if _, err := os.Stat(ShootKubeconfigPath); err != nil {
		log.Fatal(errors.Wrapf(err, "file %s does not exist: ", ShootKubeconfigPath))
	}

	if CloudProvider == "" {
		CloudProvider = os.Getenv("PROVIDER_TYPE")
	}
	if CloudProvider == "" {
		log.Fatal("PROVIDER_TYPE environment variable not found")
	}

	GardenerVersion = tiutil.Getenv("GARDENER_VERSION", "")

	tmpDir := "/tmp/e2e"
	ExportPath = tiutil.Getenv("E2E_EXPORT_PATH", path.Join(tmpDir, "export"))
	if _, err := os.Stat(ExportPath); os.IsNotExist(err) {
		if err := os.MkdirAll(ExportPath, os.FileMode(0777)); err != nil {
			log.Fatal(err)
		}
	}
}

// LogConfig sends the most relevant configuration options and their values to its logger
func LogConfig(log logr.Logger) {
	log.Info("Running with configuration",
		"ExportPath", ExportPath,
		"ConformanceLogLevel", ConformanceLogLevel,
		"K8sRelease", K8sRelease,
		"HydrophoneVersion", HydrophoneVersion,
		"GinkgoParallel", GinkgoParallel,
		"FlakeAttempts", FlakeAttempts,
		"PublishResultsToTestgrid", PublishResultsToTestgrid,
		"SkipIndividualTestCases", SkipIndividualTestCases,
	)
	if PublishResultsToTestgrid {
		log.Info("Test results will be uploaded to:", "Project", GcsProjectID, "Bucket", GcsBucket)
	}
}
