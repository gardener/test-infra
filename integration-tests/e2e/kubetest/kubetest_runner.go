package kubetest

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/gardener/test-infra/integration-tests/e2e/config"
	"github.com/gardener/test-infra/integration-tests/e2e/util"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

type TestsKind string

const (
	logMaxLength = 300
)

func init() {
	var err error
	kubectlPath, _ := exec.Command("which", "kubectl").Output() // error is checked in previous steps
	if err := os.Setenv("KUBECTL_PATH", strings.TrimSpace(string(kubectlPath))); err != nil {
		log.Fatal(errors.Wrapf(err, "couldn't set environment variable KUBECTL_PATH"))
	}
	log.Debugf("KUBECTL_PATH: '%s'", os.Getenv("KUBECTL_PATH"))

	if err = os.Setenv("KUBECONFIG", config.ShootKubeconfigPath); err != nil {
		log.Fatal(errors.Wrapf(err, "couldn't set environment variable KUBECONFIG"))
	}
	log.Debugf("KUBECONFIG: '%s'", os.Getenv("KUBECONFIG"))

	if err = os.Setenv("KUBERNETES_CONFORMANCE_TEST", "y"); err != nil {
		log.Fatal(errors.Wrapf(err, "couldn't set environment variable KUBERNETES_CONFORMANCE_TEST"))
	}

	if err = os.Setenv("GINKGO_NO_COLOR", "y"); err != nil {
		log.Fatal(errors.Wrapf(err, "couldn't set environment variable GINKGO_NO_COLOR"))
	}
}

// DryRun runs kubetest with dryrun=true argument
func DryRun() (logDir string) {
	kubetestArgs := createKubetestArgs("", false, true, 1)
	runKubetest(kubetestArgs, false)
	return kubetestArgs.LogDir
}

// Run runs kubetest to execute e2e testcases for a given testcase description file
func Run(descFile string) (resultsPath string) {
	if descFile == "" {
		log.Fatal("no valid description file provided.")
	}
	log.Infof("running kubetest for %d e2e tests:", getLinesCount(descFile))

	parallelTestsFocus, serialTestsFocus := escapeAndConcat(descFile)
	if serialTestsFocus != "" {
		kubtestArgs := createKubetestArgs(serialTestsFocus, false, false, config.FlakeAttempts)
		log.Info("run kubetest in serial way")
		log.Infof("kubetest dump dir: %s", kubtestArgs.LogDir)
		runKubetest(kubtestArgs, false)
	}
	if parallelTestsFocus != "" {
		kubtestArgs := createKubetestArgs(parallelTestsFocus, true, false, config.FlakeAttempts)
		log.Info("run kubetest in parallel way")
		log.Infof("kubetest dump dir: %s", kubtestArgs.LogDir)
		runKubetest(kubtestArgs, false)
	}
	return config.LogDir
}

func getFileContent(filepath string) string {
	if file, err := os.Open(filepath); err != nil {
		log.Fatal(err)
	} else {
		b, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatal(err)
		}
		return string(b)
	}
	return ""
}

func createKubetestArgs(ginkgoFocus string, parallel, dryRun bool, flakeAttempts int) KubetestArgs {
	ginkgoParallelArg := ""
	if parallel {
		ginkgoParallelArg = "--ginkgo-parallel=8"
	}
	if ginkgoFocus != "" {
		ginkgoFocus = fmt.Sprintf("--ginkgo.focus=%s", ginkgoFocus)
	}
	logDir := filepath.Join(config.LogDir, strconv.FormatInt(time.Now().Unix(), 10))
	_ = os.MkdirAll(logDir, os.FileMode(0777))
	kubetestArgs := KubetestArgs{ShootConfigPath: config.ShootKubeconfigPath, GinkgoFocus: ginkgoFocus, DryRun: dryRun, LogDir: logDir, FlakeAttempts: flakeAttempts, Provider: Cloudprovider(config.CloudProvider), GinkgoParallel: ginkgoParallelArg}
	return kubetestArgs
}

func runKubetest(args KubetestArgs, logToStd bool) {
	//  -clean-start
	//    	If true, purge all namespaces except default and system before running tests. This serves to Cleanup test namespaces from failed/interrupted e2e runs in a long-lived cluster.
	ginkgoArgs := fmt.Sprintf("--test_args=--ginkgo.flakeAttempts=%o --ginkgo.dryRun=%t --minStartupPods=1 %s", args.FlakeAttempts, args.DryRun, args.GinkgoFocus)
	cmd := exec.Command("kubetest", "--provider=skeleton", "--deployment=local", "--test", "--check-version-skew=false", args.GinkgoParallel, ginkgoArgs, fmt.Sprintf("--dump=%s", args.LogDir))
	cmd.Dir = config.KubernetesPath

	cmdString := strings.Join(cmd.Args, " ")
	logMsg := fmt.Sprintf("Executing '%s' in working dir '%s'", cmdString, cmd.Dir)
	if len(logMsg) > logMaxLength {
		log.Infof("%s...", logMsg[:logMaxLength])
	} else {
		log.Info(logMsg)
	}

	// setup log file
	e2eLogFilePath := filepath.Join(args.LogDir, "e2e.log")
	file, err := os.Create(e2eLogFilePath)
	if err != nil {
		log.Error(err)
	}

	cmd.Stdout = file
	if logToStd {
		outWriter := io.MultiWriter(os.Stdout, file)
		cmd.Stdout = outWriter
	}
	cmd.Stderr = os.Stderr

	if err = cmd.Start(); err != nil {
		log.Error(err)
	}
	if err = cmd.Wait(); err != nil {
		log.Error(err)
	}
	file.Close()

	// kubetest run fails if one of the testcases failes, therefore the execution is still successful and no err needs to be returned
	if err != nil {
		file, err := os.Open(e2eLogFilePath)
		defer file.Close()
		if err != nil {
			log.Error(err)
			return
		}

		bufferSize := int64(5000)
		buf := make([]byte, bufferSize)
		stat, err := os.Stat(e2eLogFilePath)
		if err != nil {
			log.Error("unable to get stat of path %s: %s", e2eLogFilePath, err.Error())
			return
		}
		var start int64
		if stat.Size() < bufferSize {
			start = 0
		} else {
			start = stat.Size() - bufferSize
		}
		_, err = file.ReadAt(buf, start)
		if err == nil || errors.Is(err, io.EOF) {
			log.Infof("BEGIN: dump kubetest stdout last %d bytes (size %d)", bufferSize, stat.Size())
			if stat.Size() > 0 {
				scanner := bufio.NewScanner(strings.NewReader(string(buf)))
				for scanner.Scan() {
					log.Info("    " + scanner.Text())
				}
			} else {
				log.Info("empty kubetest stdout")
			}
			log.Infof("END: dump kubetest stdout last %d bytes", bufferSize)
		} else {
			log.Error(errors.Wrapf(err, "kubetest run failed"))
		}

		if err = util.DumpShootLogs(config.GardenKubeconfigPath, config.ProjectNamespace, config.ShootName); err != nil {
			log.Error(errors.Wrap(err, "could not execute shoot dump"))
		}
	} else {
		log.Info("kubetest test run successful")
	}
}

func escapeAndConcat(descFilePath string) (concatenatedParallelTests, concatenatedSerialTests string) {
	var serialTestsBuffer bytes.Buffer
	var parallelTestsBuffer bytes.Buffer

	file, err := os.Open(descFilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	if config.GinkgoParallel {
		for scanner.Scan() {
			testcase := strings.TrimSpace(scanner.Text())
			if testcase == "" {
				continue
			}
			if strings.Contains(testcase, "[Serial]") || strings.Contains(testcase, "[Disruptive]") {
				if serialTestsBuffer.Len() > 0 {
					serialTestsBuffer.WriteString("|")
				}
				serialTestsBuffer.WriteString(escapeForRegex(testcase))
			} else {
				if parallelTestsBuffer.Len() > 0 {
					parallelTestsBuffer.WriteString("|")
				}
				parallelTestsBuffer.WriteString(escapeForRegex(testcase))
			}
		}
	} else {
		for scanner.Scan() {
			testcase := strings.TrimSpace(scanner.Text())
			if testcase == "" {
				continue
			}
			if serialTestsBuffer.Len() > 0 {
				serialTestsBuffer.WriteString("|")
			}
			serialTestsBuffer.WriteString(escapeForRegex(testcase))
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(errors.Wrapf(err, "scanning %s failed", descFilePath))
	}
	return parallelTestsBuffer.String(), serialTestsBuffer.String()
}

func escapeForRegex(input string) string {
	output := strings.Replace(regexp.QuoteMeta(input), " ", "\\s", -1)
	return output
}

func getLinesCount(filepath string) int {
	r, err := os.Open(filepath)
	if err != nil {
		log.Fatal(err)
	}
	buf := make([]byte, 32*1024)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := r.Read(buf)
		count += bytes.Count(buf[:c], lineSep)

		switch {
		case err == io.EOF:
			return count

		case err != nil:
			log.Warn(err)
			return count
		}
	}
}

type Cloudprovider string

type KubetestArgs struct {
	FlakeAttempts   int
	ShootConfigPath string
	DryRun          bool
	LogDir          string
	GinkgoParallel  string
	Provider        Cloudprovider
	Skip            string
	GinkgoFocus     string
}
