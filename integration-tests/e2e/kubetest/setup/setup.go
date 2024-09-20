package setup

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/gardener/test-infra/integration-tests/e2e/config"
	"github.com/gardener/test-infra/integration-tests/e2e/kubetest"
	"github.com/gardener/test-infra/integration-tests/e2e/util"
	tmutil "github.com/gardener/test-infra/pkg/util"
	k8sutil "k8s.io/test-infra/kubetest/util"
)

func Setup() error {
	cleanUpPreviousRuns()
	if err := areTestUtilitiesReady(); err == nil {
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
	// go get does not support installation with golang 1.18 or higher.
	// However, go install is flawed in a sense that it does not support the replace directive in a go.mod file
	// see https://github.com/golang/go/issues/44840 and https://github.com/kubernetes/test-infra/issues/25950#issuecomment-1101933386 for reference
	// the "solution" seems to be to git clone and run a go build
	log.Info("Setting up kubetest binary...")
	if _, err := util.RunCmd("git clone -n https://github.com/kubernetes/test-infra.git /go/src/github.com/kubernetes/test-infra/", "/"); err != nil {
		return err
	}
	if _, err := util.RunCmd(fmt.Sprintf("git checkout %s", config.TestInfraVersion), "/go/src/github.com/kubernetes/test-infra/"); err != nil {
		return err
	}
	if _, err := util.RunCmd("go build -o /go/bin", "/go/src/github.com/kubernetes/test-infra/kubetest"); err != nil {
		return err
	}

	downloadRoot := k8sutil.K8s("kubernetes", "_output", "gcs-stage")
	downloadFolderVersion := fmt.Sprintf("%s/v%s", downloadRoot, config.K8sRelease)

	log.Infof("Creating K8s directory at %s ...", downloadFolderVersion)
	err := os.MkdirAll(downloadFolderVersion, 700)
	if err != nil {
		return err
	}

	downloadUrlBase := fmt.Sprintf("https://dl.k8s.io/v%s", config.K8sRelease)
	fileNames := []string{
		"kubernetes.tar.gz",
		"kubernetes-client-linux-amd64.tar.gz",
		"kubernetes-server-linux-amd64.tar.gz",
		"kubernetes-test-linux-amd64.tar.gz",
		"kubernetes-test-portable.tar.gz",
	}

	for _, resource := range fileNames {

		downloadURL := fmt.Sprintf("%s/%s", downloadUrlBase, resource)
		log.Infof("Downloading K8s archive from %s ...", downloadURL)
		response, err := http.Get(downloadURL)
		if err != nil {
			return err
		}
		defer response.Body.Close()

		k8sFilePath := fmt.Sprintf("%s/%s", downloadFolderVersion, resource)
		log.Infof("Create archive file at %s ...", k8sFilePath)
		f, err := os.Create(k8sFilePath)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(f, response.Body)
		if err != nil {
			return err
		}
	}
	log.Info("Finished downloading and storing K8s archive for kubetest")

	log.Info("Setting up tools...")
	if err := os.MkdirAll(config.K8sRoot, os.ModePerm); err != nil {
		return errors.Wrapf(err, "unable to create directories %s", config.K8sRoot)
	}

	if _, err := util.RunCmd(fmt.Sprintf("kubetest --provider=skeleton --extract=local"), config.K8sRoot); err != nil {
		return err
	}
	if err := cleanUpLargeUnsedFiles(); err != nil {
		return err
	}
	return nil
}

func cleanUpLargeUnsedFiles() error {
	// at this point all archive files should have been extracted and therefore are not needed anymore
	unusedFiles := util.GetFilesByPattern(config.K8sRoot, "tar.gz$")
	// delete unused files / directories
	for _, unusedFile := range unusedFiles {
		log.Infof("Removing unused directory/file: %s", unusedFile)
		if err := os.RemoveAll(unusedFile); err != nil {
			return err
		}
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
	} else {
		log.Info("kubetest binary available")
	}

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
	currentKubernetesVersionByte, err := os.ReadFile(kubernetesVersionFile)
	if err != nil || len(currentKubernetesVersionByte) == 0 {
		res = multierror.Append(res, fmt.Errorf("Required file %s does not exist or is empty: ", kubernetesVersionFile))
	} else if currentKubernetesVersion := strings.TrimSpace(string(currentKubernetesVersionByte[1:])); currentKubernetesVersion != config.K8sRelease {
		res = multierror.Append(res, fmt.Errorf("found kubernetes version %s, required version %s: ", currentKubernetesVersion, config.K8sRelease))
	}

	return tmutil.ReturnMultiError(res)
}
