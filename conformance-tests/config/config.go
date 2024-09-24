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
	ConformanceLogLevel     int
	HydrophoneVersion       string
	K8sRelease              string
	SkipIndividualTestCases string
	GinkgoParallel          bool
	FlakeAttempts           int
	ExportPath              string
	ShootKubeconfigPath     string
	GardenKubeconfigPath    string
	ProjectNamespace        string
	ShootName               string
	CloudProvider           string
	DryRun                  bool
)

func init() {
	flag.IntVar(&ConformanceLogLevel, "conformanceLogLevel", 2, "Hydrophone log level for conformance test")
	flag.StringVar(&HydrophoneVersion, "hydrophoneVersion", "", "Hydrophone version to be used")
	flag.StringVar(&K8sRelease, "k8sVersion", "", "Kubernetes release version e.g. 1.14.0")
	flag.StringVar(&SkipIndividualTestCases, "skipIndividualTestCases", "", "A list of ginkgo.skip patterns (regex based) to skip individual test cases, use \"|\" as delimiter.")
	flag.IntVar(&FlakeAttempts, "flakeAttempts", 1, "Testcase flake attempts. Will run testcase n times, until it is successful")
	flag.StringVar(&ShootKubeconfigPath, "kubeconfig", "", "Kubeconfig file path of cluster to test")
	flag.StringVar(&GardenKubeconfigPath, "gardenKubeconfig", "", "kubeconfig file path of the virtual garden cluster")
	flag.StringVar(&ShootName, "shoot", "", "name of the shoot cluster")
	flag.StringVar(&ProjectNamespace, "project", "", "name of the garden project")
	flag.StringVar(&CloudProvider, "cloudprovider", "", "Cluster cloud provider (aws, gcp, azure, alicloud, openstack)")
	flag.BoolVar(&DryRun, "dryRun", false, "use in combination with --conformanceLogLevel to output testcases")
	// publish??

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

	if FlakeAttempts == 0 {
		log.Fatal("flakeAttempts of 0 zero doesn't make sense. Use >= 1 to have at least 1 execution.")
	}

	if ShootKubeconfigPath == "" {
		ShootKubeconfigPath = tiutil.Getenv("E2E_KUBECONFIG_PATH", os.Getenv("KUBECONFIG"))
	}
	if ShootKubeconfigPath == "" {
		log.Fatal("shoot config not set")
	}
	if _, err := os.Stat(ShootKubeconfigPath); err != nil {
		log.Fatal(errors.Wrapf(err, "file %s does not exist: ", ShootKubeconfigPath))
	}

	if GardenKubeconfigPath == "" {
		GardenKubeconfigPath = os.Getenv("GARDEN_KUBECONFIG_PATH")
	}
	if _, err := os.Stat(GardenKubeconfigPath); err != nil {
		log.Fatal(errors.Wrapf(err, "file %s does not exist: ", GardenKubeconfigPath))
	}

	if ProjectNamespace == "" {
		ProjectNamespace = os.Getenv("PROJECT_NAMESPACE")
	}

	if ShootName == "" {
		ShootName = os.Getenv("SHOOT_NAME")
	}

	if CloudProvider == "" {
		CloudProvider = os.Getenv("PROVIDER_TYPE")
	}
	if CloudProvider == "" {
		log.Fatal("PROVIDER_TYPE environment variable not found")
	}

	tmpDir := "/tmp/e2e"
	ExportPath = tiutil.Getenv("E2E_EXPORT_PATH", path.Join(tmpDir, "export"))
	if _, err := os.Stat(ExportPath); os.IsNotExist(err) {
		if err := os.MkdirAll(ExportPath, os.FileMode(0777)); err != nil {
			log.Fatal(err)
		}
	}
}

func LogConfig(log logr.Logger) {
	log.Info("Running with configuration",
		"ExportPath", ExportPath,
		"ConformanceLogLevel", ConformanceLogLevel,
		"K8sRelease", K8sRelease,
		"HydrophoneVersion", HydrophoneVersion,
		"GinkgoParallel", GinkgoParallel,
		"FlakeAttempts", FlakeAttempts,
		"SkipIndividualTestCases", SkipIndividualTestCases,
	)
}
