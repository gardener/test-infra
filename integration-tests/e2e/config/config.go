package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	flag "github.com/spf13/pflag"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	tiutil "github.com/gardener/test-infra/pkg/util"
)

type arrayTestcase []string

func (i *arrayTestcase) Type() string {
	return "TestCasesArray"
}

func (i *arrayTestcase) String() string {
	return fmt.Sprintf("%s", *i)
}

func (i *arrayTestcase) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	GoPath                   string
	K8sRoot                  string
	KubernetesPath           string
	TestInfraPath            string
	ExportPath               string
	OwnDir                   string
	LogDir                   string
	TmpDir                   string
	ShootKubeconfigPath      string
	GardenKubeconfigPath     string
	ProjectNamespace         string
	ShootName                string
	GinkgoParallel           bool
	DescriptionFile          string
	K8sRelease               string
	CloudProvider            string
	IgnoreFalsePositiveList  bool
	IncludeUntrackedTests    bool
	DescriptionFilePath      string
	FlakeAttempts            int
	PublishResultsToTestgrid bool
	DescriptionsPath         string
	IgnoreSkipList           bool
	K8sReleaseMajorMinor     string
	GardenerVersion          string
	RetestFlaggedOnly        bool
	TestcaseGroup            []string
	TestcaseGroupString      string
	ExplicitTestcases        arrayTestcase
	DownloadsDir             string
	RunCleanUpAfterTest      bool
	RetryFailedTestcases     bool
	DryRun                   bool
)

const (
	// points to a commit in https://github.com/kubernetes/test-infra/commits/master
	TestInfraVersion = "0fdffe60e81f1a3e22d5c1ae86e9e2631c5eb96a"
)

var Debug bool

const (
	workingDescFile = "working.json"
)

func init() {
	flag.BoolVar(&Debug, "debug", false, "Run e2e in debug mode")
	flag.BoolVar(&RunCleanUpAfterTest, "cleanUpAfterwards", false, "Clean downloads folder and remove kubernetes folder after test run")
	flag.StringVar(&ShootKubeconfigPath, "kubeconfig", "", "Kubeconfig file path of cluster to test")
	flag.StringVar(&GardenKubeconfigPath, "gardenKubeconfig", "", "kubeconfig file path of the virtual garden cluster")
	flag.StringVar(&ShootName, "shoot", "", "name of the shoot cluster")
	flag.StringVar(&ProjectNamespace, "project", "", "name of the garden project")
	flag.StringVar(&K8sRelease, "k8sVersion", "", "Kubernetes release version e.g. 1.14.0")
	flag.StringVar(&CloudProvider, "cloudprovider", "", "Cluster cloud provider (aws, gcp, azure, alicloud, openstack)")
	flag.IntVar(&FlakeAttempts, "flakeAttempts", 1, "Testcase flake attempts. Will run testcase n times, until it is successful")
	flag.StringVar(&TestcaseGroupString, "testcasegroup", "", "Testcase groups to run (conformance, fast, slow")
	flag.Var(&ExplicitTestcases, "testcase", "List of testcases. If used description file and execution group are ingored.")
	flag.BoolVar(&RetryFailedTestcases, "retryFailedTestcases", false, "runs an additional kubetest run for failed tests only")
	flag.BoolVar(&DryRun, "dryRun", false, "specify dryRun = true to only display all testcases")
	flag.Parse()
	if Debug {
		log.SetLevel(log.DebugLevel)
	}

	_, b, _, _ := runtime.Caller(0)
	OwnDir = filepath.Dir(filepath.Dir(b))
	TmpDir = "/tmp/e2e/"
	LogDir = path.Join(TmpDir, "artifacts")
	DownloadsDir = path.Join(TmpDir, "downloads")
	_ = os.Mkdir(LogDir, os.FileMode(0777))
	_ = os.Mkdir(TmpDir, os.FileMode(0777))
	_ = os.Mkdir(DownloadsDir, os.FileMode(0777))
	GoPath = os.Getenv("GOPATH")
	if GoPath == "" {
		log.Fatal("GOPATH environment variable not found")
	}
	GardenerVersion = tiutil.Getenv("GARDENER_VERSION", "")
	ExportPath = tiutil.Getenv("E2E_EXPORT_PATH", path.Join(TmpDir, "export"))
	if _, err := os.Stat(ExportPath); os.IsNotExist(err) {
		if err := os.MkdirAll(ExportPath, os.FileMode(0777)); err != nil {
			log.Fatal(err)
		}
	}

	K8sRoot = filepath.Join(GoPath, "src/k8s.io")
	KubernetesPath = filepath.Join(K8sRoot, "kubernetes")
	TestInfraPath = filepath.Join(K8sRoot, "test-infra")
	if ShootKubeconfigPath == "" {
		ShootKubeconfigPath = tiutil.Getenv("E2E_KUBECONFIG_PATH", os.Getenv("KUBECONFIG"))
	}
	if ShootKubeconfigPath == "" {
		log.Fatal("shoot config not set")
	}
	if GardenKubeconfigPath == "" {
		GardenKubeconfigPath = os.Getenv("GARDEN_KUBECONFIG_PATH")
	}
	if ProjectNamespace == "" {
		ProjectNamespace = os.Getenv("PROJECT_NAMESPACE")
	}
	if ShootName == "" {
		ShootName = os.Getenv("SHOOT_NAME")
	}
	if _, err := os.Stat(ShootKubeconfigPath); err != nil {
		log.Fatal(errors.Wrapf(err, "file %s does not exist: ", ShootKubeconfigPath))
	}
	GinkgoParallel = tiutil.GetenvBool("GINKGO_PARALLEL", true)
	DescriptionFile = tiutil.Getenv("DESCRIPTION_FILE", workingDescFile)
	if K8sRelease == "" {
		K8sRelease = os.Getenv("K8S_VERSION")
	}
	if K8sRelease == "" {
		log.Fatal("K8S_VERSION environment variable not found")
	}
	if len(ExplicitTestcases) != 0 {
		TestcaseGroupString = "explicit"
	}
	if TestcaseGroupString == "" {
		TestcaseGroup = strings.Split(os.Getenv("TESTCASE_GROUPS"), ",")
	} else {
		TestcaseGroup = strings.Split(TestcaseGroupString, ",")
	}
	sort.Strings(TestcaseGroup)
	if len(TestcaseGroup) == 0 {
		log.Fatal("TESTCASE_GROUP environment variable not found")
	}
	if CloudProvider == "" {
		CloudProvider = os.Getenv("PROVIDER_TYPE")
	}
	if CloudProvider == "" {
		log.Fatal("PROVIDER_TYPE environment variable not found")
	}
	IgnoreFalsePositiveList = tiutil.GetenvBool("IGNORE_FALSE_POSITIVE_LIST", false)
	IncludeUntrackedTests = tiutil.GetenvBool("INCLUDE_UNTRACKED_TESTS", false)
	K8sReleaseMajorMinor = string(regexp.MustCompile(`^(\d+\.\d+)`).FindSubmatch([]byte(K8sRelease))[1])
	DescriptionsPath = path.Join(OwnDir, "kubetest", "description", K8sReleaseMajorMinor)
	DescriptionFilePath = path.Join(DescriptionsPath, DescriptionFile)
	if _, err := os.Stat(DescriptionFilePath); err != nil {
		log.Fatal(errors.Wrapf(err, "file %s does not exist: ", DescriptionFilePath))
	}
	if FlakeAttempts == 0 {
		log.Fatal("flakeAttempts of 0 zero doesn't make sense. Use >= 1 to have at least 1 execution.")
	}
	PublishResultsToTestgrid = tiutil.GetenvBool("PUBLISH_RESULTS_TO_TESTGRID", false)
	IgnoreSkipList = tiutil.GetenvBool("IGNORE_SKIP_LIST", false)
	RetestFlaggedOnly = tiutil.GetenvBool("RETEST_FLAGGED_ONLY", false)

	log.Debugf("GoPath: %s", GoPath)
	log.Debugf("K8sRoot: %s", K8sRoot)
	log.Debugf("KubernetesPath: %s", KubernetesPath)
	log.Debugf("OwnDir: %s", OwnDir)
	log.Debugf("LogDir: %s", LogDir)
	log.Debugf("ExportPath: %s", ExportPath)
	log.Debugf("TestInfraPath: %s", TestInfraPath)
	log.Debugf("ShootKubeconfigPath: %s", ShootKubeconfigPath)
	log.Debugf("GinkgoParallel: %t", GinkgoParallel)
	log.Debugf("K8sRelease: %s", K8sRelease)
	log.Debugf("CloudProvider: %s", CloudProvider)
	log.Debugf("IgnoreFalsePositiveList: %t", IgnoreFalsePositiveList)
	log.Debugf("IncludeUntrackedTests: %t", IncludeUntrackedTests)
	log.Debugf("DescriptionFile: %s", DescriptionFile)
	log.Debugf("DescriptionFilePath: %s", DescriptionFilePath)
	log.Debugf("IgnoreSkipList: %t", IgnoreSkipList)
	log.Debugf("PublishResultsToTestgrid: %t", PublishResultsToTestgrid)
	log.Debugf("FlakeAttempts: %o", FlakeAttempts)
	log.Debugf("GardenerVersion: %s", GardenerVersion)
	log.Debugf("RetestFlaggedOnly: %t", RetestFlaggedOnly)
	log.Debugf("TestcaseGroup: %v", TestcaseGroup)
	log.Debugf("ExplicitTestcases: %v", strings.Join(ExplicitTestcases, ", "))
}
