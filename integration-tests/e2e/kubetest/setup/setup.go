package setup

import (
	"fmt"
	"github.com/gardener/test-infra/integration-tests/e2e/config"
	"github.com/gardener/test-infra/integration-tests/e2e/kubetest"
	"github.com/gardener/test-infra/integration-tests/e2e/util"
	tmutil "github.com/gardener/test-infra/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func Setup() error {
	cleanUpPreviousRuns()
	if err := areTestUtilitiesReady(); err != nil {
		log.Info("all test utilities were already ready")
		log.Info("setup finished successfuly. Testutilities ready. Kubetest is ready for usage.")
		return nil
	}

	log.Info("test utilities are not ready. Install...")
	if err := getKubetestAndUtilities(); err != nil {
		return errors.Wrap(err, "unable to setup kubetest and utilities")
	}

	if err := areTestUtilitiesReady(); err != nil {
		return err
	}
	log.Info("setup finished successfuly. Testutilities ready. Kubetest is ready for usage.")
	return nil
}

func getKubetestAndUtilities() error {
	goModuleOriginValue := os.Getenv("GO111MODULE")
	_ = os.Setenv("GO111MODULE", "on")
	if _, err := util.RunCmd("go get k8s.io/test-infra/kubetest", ""); err != nil {
		return err
	}
	_ = os.Setenv("GO111MODULE", goModuleOriginValue)

	if err := os.MkdirAll(config.K8sRoot, os.ModePerm); err != nil {
		return errors.Wrapf(err, "unable to create directories %s", config.K8sRoot)
	}
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

func areTestUtilitiesReady() error {
	log.Info("checking whether any test utility is not ready")
	var res *multierror.Error

	if !util.CommandExists("kubetest") {
		res = multierror.Append(res, errors.New("kubetest not installed"))
	}
	log.Info("kubetest binary available")

	// check if required directories exist
	requiredPaths := [...]string{
		path.Join(config.K8sRoot, "kubernetes/hack"),
		path.Join(config.K8sRoot, "kubernetes/cluster"),
		path.Join(config.K8sRoot, "kubernetes/test"),
		path.Join(config.K8sRoot, "kubernetes/client"),
		path.Join(config.K8sRoot, "kubernetes/server")}
	for _, requiredPath := range requiredPaths {
		if _, err := os.Stat(requiredPath); err != nil {
			res = multierror.Append(res, errors.Wrapf(err, "dir %s does not exist: ", requiredPath))
		} else {
			log.Info(fmt.Sprintf("%s dir exists", requiredPath))
		}
	}

	kubernetesVersionFile := path.Join(config.K8sRoot, "kubernetes/version")
	currentKubernetesVersionByte, err := ioutil.ReadFile(kubernetesVersionFile)
	if err != nil || len(currentKubernetesVersionByte) == 0 {
		res = multierror.Append(res, fmt.Errorf("Required file %s does not exist or is empty: ", kubernetesVersionFile))
	} else if currentKubernetesVersion := strings.TrimSpace(string(currentKubernetesVersionByte[1:])); currentKubernetesVersion != config.K8sRelease {
		res = multierror.Append(res, fmt.Errorf("found kubernetes version %s, required version %s: ", currentKubernetesVersion, config.K8sRelease))
	}
	return tmutil.ReturnMultiError(res)
}
