package config

import (
	"github.com/gardener/test-infra/test/e2etest/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

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
)

var WORKING_DESC_FILE = "working.json"

func init() {
	//log.SetLevel(log.DebugLevel)

	_, b, _, _ := runtime.Caller(0)
	OwnDir = filepath.Dir(filepath.Dir(b))
	TmpDir = "/tmp/e2e/"
	LogDir = path.Join(TmpDir, "artifacts")
	_ = os.Mkdir(LogDir, os.FileMode(0777))
	_ = os.Mkdir(TmpDir, os.FileMode(0777))
	GoPath = os.Getenv("GOPATH")
	if GoPath == "" {
		log.Fatal("GOPATH environment variable not found")
	}
	GardenerVersion = util.GetEnv("GARDENER_VERSION", "")
	ExportPath = util.GetEnv("EXPORT_PATH", path.Join(TmpDir, "export"))
	if _, err := os.Stat(ExportPath); os.IsNotExist(err) {
		if err := os.MkdirAll(ExportPath, os.FileMode(0777)); err != nil {
			log.Fatal(err)
		}
	}

	K8sRoot = filepath.Join(GoPath, "src/k8s.io")
	KubernetesPath = filepath.Join(K8sRoot, "kubernetes")
	TestInfraPath = filepath.Join(K8sRoot, "test-infra")
	ShootKubeconfigPath = filepath.Join(ExportPath, "shoot.config")
	if _, err := os.Stat(ShootKubeconfigPath); err != nil {
		log.Fatal(errors.Wrapf(err, "file %s does not exist: ", ShootKubeconfigPath))
	}
	GinkgoParallel, _ = strconv.ParseBool(util.GetEnv("GINKGO_PARALLEL", "true"))
	DescriptionFile = util.GetEnv("DESCRIPTION_FILE", WORKING_DESC_FILE)
	K8sRelease = os.Getenv("K8S_VERSION")
	if K8sRelease == "" {
		log.Fatal("K8S_VERSION environment variable not found")
	}
	TestcaseGroup = strings.Split(os.Getenv("TESTCASE_GROUPS"), ",")
	sort.Strings(TestcaseGroup)
	if len(TestcaseGroup) == 0 {
		log.Fatal("TESTCASE_GROUP environment variable not found")
	}
	CloudProvider = os.Getenv("CLOUDPROVIDER")
	if CloudProvider == "" {
		log.Fatal("CLOUDPROVIDER environment variable not found")
	}
	IgnoreFalsePositiveList, _ = strconv.ParseBool(util.GetEnv("IGNORE_FALSE_POSITIVE_LIST", "false"))
	IncludeUntrackedTests, _ = strconv.ParseBool(util.GetEnv("INCLUDE_UNTRACKED_TESTS", "false"))
	K8sReleaseMajorMinor = string(regexp.MustCompile(`^(\d+\.\d+)`).FindSubmatch([]byte(K8sRelease))[1])
	DescriptionsPath = path.Join(OwnDir, "kubetest", "description", K8sReleaseMajorMinor)
	DescriptionFilePath = path.Join(DescriptionsPath, DescriptionFile)
	if _, err := os.Stat(DescriptionFilePath); err != nil {
		log.Fatal(errors.Wrapf(err, "file %s does not exist: ", DescriptionFilePath))
	}
	FlakeAttempts, _ = strconv.Atoi(util.GetEnv("FLAKE_ATTEMPTS", "2"))
	PublishResultsToTestgrid, _ = strconv.ParseBool(util.GetEnv("PUBLISH_RESULTS_TO_TESTGRID", "false"))
	IgnoreSkipList, _ = strconv.ParseBool(util.GetEnv("IGNORE_SKIP_LIST", "false"))
	RetestFlaggedOnly, _ = strconv.ParseBool(util.GetEnv("RETEST_FLAGGED_ONLY", "false"))

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
	log.Debugf("GardenerVersion: %o", GardenerVersion)
	log.Debugf("RetestFlaggedOnly: %o", RetestFlaggedOnly)
	log.Debugf("TestcaseGroup: %o", TestcaseGroup)
}
