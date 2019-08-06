package setup

import (
	"fmt"
	"github.com/gardener/test-infra/integration-tests/e2e/config"
	"github.com/gardener/test-infra/integration-tests/e2e/kubetest"
	"github.com/gardener/test-infra/integration-tests/e2e/util"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var k8sOutputBinDir string = filepath.Join(config.KubernetesPath, "_output/bin")

func Setup() error {
	cleanUpPreviousRuns()
	if areTestUtilitiesReady() {
		log.Info("all test utilities were already ready")
		return nil
	}

	log.Info("test utilities are not ready. Install...")
	if err := getKubetestAndUtilities(); err != nil {
		return err
	}
	log.Info("setup finished successfuly. Testutilities ready. Kubetest is ready for usage.")
	return nil
}

func getKubetestAndUtilities() error {
	goModuleOriginValue := os.Getenv("GO111MODULE")
	_ = os.Setenv("GO111MODULE", "off")
	if _, err := util.RunCmd("go get k8s.io/test-infra/kubetest", ""); err != nil {
		return err
	}
	_ = os.Setenv("GO111MODULE", goModuleOriginValue)
	if _, err := util.RunCmd(fmt.Sprintf("kubetest --provider=skeleton --extract=v%s", config.K8sRelease), config.K8sRoot); err != nil {
		return err
	}
	return nil
}

func cleanUpPreviousRuns() {
	if err := os.RemoveAll(config.LogDir); err != nil {
		log.Error(err)
	}
	testResultFiles := util.GetFilesByPattern(config.ExportPath, `test.*\.json$`)
	for _, file := range testResultFiles {
		if err := os.Remove(file); err != nil {
			log.Error(err)
		}
	}
	if err := os.Remove(kubetest.GeneratedRunDescPath); err != nil {
		log.Error(err)
	}
	_ = os.Remove(filepath.Join(config.ExportPath, "started.json"))
	_ = os.Remove(filepath.Join(config.ExportPath, "finished.json"))
	_ = os.Remove(filepath.Join(config.ExportPath, "e2e.log"))
	_ = os.Remove(filepath.Join(config.ExportPath, "junit_01.xml"))
}

func PostRunCleanFiles() error {
	// remove log dir
	if err := os.RemoveAll(config.LogDir); err != nil {
		return err
	}
	// remove kubernetes folder
	if err := os.RemoveAll(os.Getenv("GOPATH")); err != nil {
		return err
	}
	//remove downloads dir
	if err := os.RemoveAll(config.DownloadsDir); err != nil {
		return err
	}
	return nil
}

func areTestUtilitiesReady() bool {
	log.Info("checking whether any test utility is not ready")
	if _, err := util.RunCmd("which kubectl", ""); err != nil {
		log.Warn("kubectl not installed")
		return false
	}
	if out, err := util.RunCmd("kubectl version", ""); err != nil || !strings.Contains(out.StdOut, fmt.Sprintf("v%s", config.K8sRelease)) {
		log.Warn("kubectl version doesn't match kubernetes version")
		return false
	}
	e2eTestPath := path.Join(k8sOutputBinDir, "e2e.test")
	if _, err := os.Stat(e2eTestPath); os.IsNotExist(err) {
		log.Warnf("test utility not ready: %s doesn't exist", e2eTestPath)
		return false // path does not exist
	}
	ginkgoPath := path.Join(k8sOutputBinDir, "ginkgo")
	if _, err := os.Stat(ginkgoPath); os.IsNotExist(err) {
		log.Warnf("test utility not ready: %s doesn't exist", ginkgoPath)
		return false // path does not exist
	}
	if out, err := util.RunCmd("git describe", config.KubernetesPath); err != nil {
		log.Warnf("failed to run 'git describe' in %s", config.KubernetesPath, err)
		return false
	} else if strings.TrimSpace(out.StdOut) != fmt.Sprintf("v%s", config.K8sRelease) {
		log.Infof("test utility not ready: current k8s release version is %s, but the requested version is v%s", out.StdOut, config.K8sRelease)
		return false
	}

	return true
}

func isNoGoFilesErr(s string) bool {
	if strings.Contains(s, "no Go files in") {
		return true
	}
	return false
}
