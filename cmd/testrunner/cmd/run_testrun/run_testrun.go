// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package run_testrun

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"os"
	"time"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/spf13/cobra"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var (
	tmKubeconfigPath     string
	namespace            string
	timeout              int64
	testrunFlakeAttempts int
	serial               bool
	backoffBucket        int
	backoffPeriod        time.Duration

	testrunPath       string
	testrunNamePrefix string
)

// AddCommand adds run-testrun to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(runTestrunCmd)
}

var runTestrunCmd = &cobra.Command{
	Use:   "run-testrun",
	Short: "Run the testrunner with a testrun",
	Aliases: []string{
		"run-tr",
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger.Log.Info("start testmachinery testrunner")
		stopCh := make(chan struct{})
		defer close(stopCh)

		w, err := testrunner.StartWatchController(logger.Log, tmKubeconfigPath, stopCh)
		if err != nil {
			logger.Log.Error(err, "unable to start testrun watch controller")
			os.Exit(1)
		}

		config := &testrunner.Config{
			Watch:         w,
			Namespace:     namespace,
			Timeout:       time.Duration(timeout) * time.Second,
			FlakeAttempts: testrunFlakeAttempts,
			ExecutorConfig: testrunner.ExecutorConfig{
				Serial:        false,
				BackoffBucket: backoffBucket,
				BackoffPeriod: backoffPeriod,
			},
		}

		tr, err := testmachinery.ParseTestrunFromFile(testrunPath)
		if err != nil {
			logger.Log.Error(err, "unable to parse testrun")
			os.Exit(1)
		}

		run := testrunner.Run{
			Testrun:  tr,
			Metadata: &metadata.Metadata{},
		}

		run.Exec(logger.Log.WithName("execute"), config, testrunNamePrefix)
		if run.Error != nil {
			logger.Log.Error(run.Error, "testrunner execution disrupted")
			os.Exit(1)
		}

		if run.Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess {
			logger.Log.Info("testrunner successfully finished.")
		} else {
			logger.Log.Error(nil, "testrunner finished unsuccessful", "phase", run.Testrun.Status.Phase)
		}

		fmt.Print(util.PrettyPrintStruct(run.Testrun.Status))
	},
}

func init() {
	// configuration flags
	runTestrunCmd.Flags().StringVar(&tmKubeconfigPath, "tm-kubeconfig-path", "", "Path to the testmachinery cluster kubeconfig")
	if err := runTestrunCmd.MarkFlagRequired("tm-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "tm-kubeconfig-path")
	}
	if err := runTestrunCmd.MarkFlagFilename("tm-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "tm-kubeconfig-path")
	}
	runTestrunCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace where the testrun should be deployed.")

	runTestrunCmd.Flags().Int64Var(&timeout, "timeout", 3600, "Timout in seconds of the testrunner to wait for the complete testrun to finish.")
	runTestrunCmd.Flags().Int64("interval", 20, "[DEPRECATED] Value has no effect")
	runTestrunCmd.Flags().IntVar(&testrunFlakeAttempts, "testrun-flake-attempts", 0, "Max number of testruns until testrun is successful")
	runTestrunCmd.Flags().BoolVar(&serial, "serial", false, "executes all testruns of a bucket only after the previous bucket has finished")
	runTestrunCmd.Flags().IntVar(&backoffBucket, "backoff-bucket", 0, "Number of parallel created testruns per backoff period")
	runTestrunCmd.Flags().DurationVar(&backoffPeriod, "backoff-period", 0, "Time to wait between the creation of testrun buckets")

	// parameter flags
	runTestrunCmd.Flags().StringVarP(&testrunPath, "file", "f", "", "Path to the testrun yaml")
	if err := runTestrunCmd.MarkFlagRequired("file"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "file")
	}
	if err := runTestrunCmd.MarkFlagFilename("file"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "file")
	}
	runTestrunCmd.Flags().StringVar(&testrunNamePrefix, "name-prefix", "testrunner-", "Name prefix of the testrun")

}
