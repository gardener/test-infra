package setup

import (
	"context"
	"fmt"
	"github.com/codeclysm/extract"
	"github.com/gardener/test-infra/integration-tests/e2e/config"
	"github.com/gardener/test-infra/integration-tests/e2e/kubetest"
	"github.com/gardener/test-infra/integration-tests/e2e/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
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
	if err := downloadKubernetes(config.K8sRelease); err != nil {
		return err
	}
	if err := downloadKubectl(config.K8sRelease); err != nil {
		return err
	}
	if err := compileOrGetTestUtilities(config.K8sRelease); err != nil {
		return err
	}
	log.Info("setup finished successfuly. Testutilities ready. Kubetest is ready for usage.")
	return nil
}

func downloadKubectl(k8sVersion string) error {
	log.Info("download corresponding kubectl version")
	if _, err := util.RunCmd(fmt.Sprintf("curl -LO https://storage.googleapis.com/kubernetes-release/release/v%s/bin/darwin/amd64/kubectl", k8sVersion), ""); err != nil {
		return err
	}
	if err := os.Chmod("./kubectl",0755); err != nil {
		return err
	}
	out, err := util.RunCmd("which kubectl", "")
	if err != nil {
		return err
	}
	if err := os.Rename("./kubectl", strings.TrimSpace(out.StdOut)); err != nil {
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

func areTestUtilitiesReady() bool {
	log.Info("checking whether any test utility is not ready")
	if _, err := exec.Command("which", "kubectl").Output(); err != nil {
		log.Fatal(errors.Wrapf(err, "kubectl command unknown"))
	}
	e2eTestPath := path.Join(k8sOutputBinDir, "e2e.test")
	if _, err := os.Stat(e2eTestPath); os.IsNotExist(err) {
		log.Infof("test utility not ready: %s doesn't exist", e2eTestPath)
		return false // path does not exist
	}
	ginkgoPath := path.Join(k8sOutputBinDir, "ginkgo")
	if _, err := os.Stat(ginkgoPath); os.IsNotExist(err) {
		log.Infof("test utility not ready: %s doesn't exist", ginkgoPath)
		return false // path does not exist
	}
	if out, err := util.RunCmd("git describe", config.KubernetesPath); err != nil {
		log.Errorf("failed to run 'git describe' in %s", config.KubernetesPath, err)
		return false
	} else if strings.TrimSpace(out.StdOut) != fmt.Sprintf("v%s", config.K8sRelease) {
		log.Infof("test utility not ready: current k8s release version is %s, but the requested version is v%s", out.StdOut, config.K8sRelease)
		return false
	}

	return true
}

func downloadKubernetes(k8sVersion string) error {
	log.Infof("get kubernetes v%s", k8sVersion)

	if _, err := os.Stat(config.KubernetesPath); !os.IsNotExist(err) {
		if _, err := util.RunCmd("git checkout master", config.KubernetesPath); err != nil {
			log.Errorf("failed to checkout master branch in %s", config.KubernetesPath, err)
			return err
		}
		if _, err := util.RunCmd("git pull --rebase", config.KubernetesPath); err != nil {
			log.Errorf("failed to run 'git pull --rebase' in %s", config.KubernetesPath, err)
			return err
		}
	} else if os.IsNotExist(err) {
		log.Infof("directory %s does not exist. Run go get -d k8s.io/kubernetes", config.KubernetesPath)
		if out, err := util.RunCmd("go get -d k8s.io/kubernetes", ""); err != nil && !isNoGoFilesErr(out.StdErr) {
			log.Error("failed to go get k8s.io/kubernetes", err)
			return err
		}
	} else {
		return err
	}

	if _, err := util.RunCmd(fmt.Sprintf("git checkout v%s", k8sVersion), config.KubernetesPath); err != nil {
		return err
	}
	return nil
}

func compileOrGetTestUtilities(k8sVersion string) error {
	k8sTestBinariesVersionURL := fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/v%s/kubernetes-test-%s-amd64.tar.gz", k8sVersion, runtime.GOOS)
	resp, err := http.Get(k8sTestBinariesVersionURL)

	if err != nil || resp.StatusCode != http.StatusOK || (runtime.GOOS != "linux" && runtime.GOOS != "darwin") {
		log.Info("no precompiled kubernetes test binaries available, or operating system is not linux/darwin, build e2e.test and ginkgo")
		if err := os.RemoveAll(k8sOutputBinDir); err != nil {
			return err
		}
		if _, err = util.RunCmd("make WHAT=test/e2e/e2e.test", config.KubernetesPath); err != nil {
			return err
		}
		if _, err = util.RunCmd("make WHAT=vendor/github.com/onsi/ginkgo/ginkgo", config.KubernetesPath); err != nil {
			return err
		}
	} else if resp.StatusCode == http.StatusOK {
		log.Infof("precompiled kubernetes test binaries available, download kubernetes-test-linux-amd64 for kubernetes v%s", k8sVersion)
		k8sTestBinariesTarPath, err := util.DownloadFile(k8sTestBinariesVersionURL, config.TmpDir)
		if err != nil {
			return err
		}

		archiveFile, err := os.Open(k8sTestBinariesTarPath)
		shiftFile := func(path string) string {
			parts := strings.Split(path, string(filepath.Separator))
			parts = parts[2:]
			return strings.Join(parts, string(filepath.Separator))
		}
		if err := extract.Gz(context.Background(), archiveFile, filepath.Dir(k8sOutputBinDir), shiftFile); err != nil {
			return err
		}

		if runtime.GOOS == "linux" {
			err = installGlibC()
		}
	}

	log.Info("get k8s examples")
	if out, err := util.RunCmd("go get -u k8s.io/examples", ""); err != nil && !isNoGoFilesErr(out.StdErr) {
		return err
	}
	return nil
}

func installGlibC() error {
	log.Info("Install glibc to run precompiled ginkgo and e2e.test binaries")
	var err error
	if _, err = util.RunCmd("apk --no-cache add ca-certificates wget", ""); err != nil {
		return err
	}
	if _, err = util.RunCmd("wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub", ""); err != nil {
		return err
	}
	if _, err = util.RunCmd("wget --quiet https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.29-r0/glibc-2.29-r0.apk", ""); err != nil {
		return err
	}
	if _, err = util.RunCmd("apk add glibc-2.29-r0.apk", ""); err != nil {
		return err
	}
	return nil
}

func isNoGoFilesErr(s string) bool {
	if strings.Contains(s, "no Go files in") {
		return true
	}
	return false
}
