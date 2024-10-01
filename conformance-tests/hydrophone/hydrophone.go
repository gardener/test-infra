// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package hydrophone

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/conformance-tests/config"
	"github.com/gardener/test-infra/pkg/util/kubernetes"
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

	hydrophoneBaseArgs := []string{
		"--kubeconfig", config.ShootKubeconfigPath,
		"-o", config.ExportPath,
		"-v", strconv.Itoa(config.ConformanceLogLevel),
	}

	hydrophoneRunArgs := []string{
		"--conformance",
		"--conformance-image", fmt.Sprintf("registry.k8s.io/conformance:v%s", config.K8sRelease),
	}

	if config.SkipIndividualTestCases != "" {
		hydrophoneRunArgs = append(hydrophoneRunArgs, "--skip", config.SkipIndividualTestCases)
	}

	if config.DryRun {
		hydrophoneRunArgs = append(hydrophoneRunArgs, "--dry-run")
	}

	// ginkgo cannot do a dry-run with multiple nodes
	if config.GinkgoParallel && !config.DryRun {
		hydrophoneRunArgs = append(hydrophoneRunArgs, "-p", "8")
	}
	if config.FlakeAttempts != 1 {
		hydrophoneRunArgs = append(hydrophoneRunArgs, "--extra-ginkgo-args", fmt.Sprintf("--flake-attempts=%d", config.FlakeAttempts))
	}

	controllerruntime.SetLogger(log)
	shootClient, err := kubernetes.NewClientFromFile(config.ShootKubeconfigPath, client.Options{})
	if err != nil {
		return err
	}

	err = runCommand(log, "hydrophone", append(hydrophoneBaseArgs, hydrophoneRunArgs...)...)
	if err != nil && isRetryable(log.WithName("IsRetryable"), shootClient) {
		for {
			log.Info("Hydrophone exited with a retryable error")
			time.Sleep(10 * time.Second)
			err = runCommand(log, "hydrophone", append(hydrophoneBaseArgs, "--continue")...)
			if err == nil || !isRetryable(log.WithName("IsRetryable"), shootClient) {
				return err
			}
		}
	}
	return err

}

// runCommand constructs a command, logs it with all arguments and executes it.
func runCommand(log logr.Logger, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	log.Info(cmd.String())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// isRetryable returns true if the conformance namespace is still active and an e2e pod exists.
func isRetryable(log logr.Logger, cl client.Client) bool {
	retries := 30
	backOff := 20 * time.Second
	for {
		conformanceNamespace := &v1.Namespace{}
		if err := cl.Get(context.Background(), client.ObjectKey{Name: "conformance"}, conformanceNamespace); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Conformance namespace does not exist anymore")
				return false
			}
			log.Error(err, "A request to the kube-apiserver of the shoot failed")
			retries--
			if retries == 0 {
				return false
			}
			time.Sleep(backOff)
			continue
		}
		if conformanceNamespace.Status.Phase != v1.NamespaceActive {
			return false
		}
		conformancePod := &v1.Pod{}
		if err := cl.Get(context.Background(), client.ObjectKey{Namespace: "conformance", Name: "e2e-conformance-test"}, conformancePod); err != nil {
			if errors.IsNotFound(err) {
				log.Info("e2e-conformance-test Pod does not exist anymore")
				return false
			}
			log.Error(err, "A request to the kube-apiserver of the shoot failed")
			retries--
			if retries == 0 {
				return false
			}
			time.Sleep(backOff)
			continue
		}
		break
	}
	return true
}
