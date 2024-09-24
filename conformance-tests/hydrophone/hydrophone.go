package hydrophone

import (
	"fmt"
	"strconv"

	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/conformance-tests/config"
	"github.com/gardener/test-infra/conformance-tests/util"
)

func Setup(log logr.Logger) error {
	moduleVersion := fmt.Sprintf("sigs.k8s.io/hydrophone@%s", config.HydrophoneVersion)
	log.Info("Setting up Hydrophone ...")
	return util.RunCommand(log, "go", "install", moduleVersion)
}

func Run(log logr.Logger) error {
	log.Info("Starting Conformance tests with Hydrophone")

	hydrophoneArgs := []string{
		"--kubeconfig", config.ShootKubeconfigPath,
		"-o", config.ExportPath,
		"--conformance",
		"-v", strconv.Itoa(config.ConformanceLogLevel),
		"--conformance-image", fmt.Sprintf("registry.k8s.io/conformance:v%s", config.K8sRelease),
	}

	if config.SkipIndividualTestCases != "" {
		hydrophoneArgs = append(hydrophoneArgs, "--skip", config.SkipIndividualTestCases)
	}
	// ginkgo cannot do a dry-run with multiple nodes
	if !config.DryRun && config.GinkgoParallel {
		hydrophoneArgs = append(hydrophoneArgs, "-p", "8")
	}
	if config.FlakeAttempts != 1 {
		hydrophoneArgs = append(hydrophoneArgs, "--extra-ginkgo-args", fmt.Sprintf("--flake-attempts=%d", config.FlakeAttempts))
	}

	err := util.RunCommand(log, "hydrophone", hydrophoneArgs...)

	return err
}
