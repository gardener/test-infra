// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package hydrophone

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/conformance-tests/config"
)

// Setup installs a given version of Hydrophone using the "go install" command
func Setup(log logr.Logger) error {
	moduleVersion := fmt.Sprintf("sigs.k8s.io/hydrophone@%s", config.HydrophoneVersion)
	log.Info("Setting up Hydrophone ...")
	return runCommand(log, "go", "install", moduleVersion)
}

// Run compiles arguments and runs K8s conformance tests using Hydrophone
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

	if config.DryRun {
		hydrophoneArgs = append(hydrophoneArgs, "--dry-run")
	}

	// ginkgo cannot do a dry-run with multiple nodes
	if config.GinkgoParallel && !config.DryRun {
		hydrophoneArgs = append(hydrophoneArgs, "-p", "8")
	}
	if config.FlakeAttempts != 1 || config.FlakeAttempts != 0 {
		hydrophoneArgs = append(hydrophoneArgs, "--extra-ginkgo-args", fmt.Sprintf("--flake-attempts=%d", config.FlakeAttempts))
	}

	return runCommand(log, "hydrophone", hydrophoneArgs...)
}

// runCommand constructs a command, logs it with all arguments and executes it.
func runCommand(log logr.Logger, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	log.Info(cmd.String())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
