// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package runtestrun

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/pkg/errors"
	"os"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/util"
	"github.com/joho/godotenv"

	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/spf13/cobra"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var (
	tmKubeconfigPath string
	namespace        string
	timeout          int64
	interval         int64

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
		logger.Log.Info("Start testmachinery testrunner")
		err := godotenv.Load()
		if err != nil {
			logger.Log.Error(err, "Error loading .env file")
		}

		tmClient, err := kubernetes.NewClientFromFile("", tmKubeconfigPath, kubernetes.WithClientOptions(client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		}))
		if err != nil {
			logger.Log.Error(err, "unable to build kubernetes client", "file", tmKubeconfigPath)
			os.Exit(1)
		}

		config := &testrunner.Config{
			TmClient:  tmClient,
			Namespace: namespace,
			Timeout:   time.Duration(timeout) * time.Second,
			Interval:  time.Duration(interval) * time.Second,
		}

		tr, err := util.ParseTestrunFromFile(testrunPath)
		if err != nil {
			logger.Log.Error(err, "unable to parse testrun")
			os.Exit(1)
		}

		run := testrunner.RunList{
			{
				Testrun:  &tr,
				Metadata: &testrunner.Metadata{},
			},
		}

		testrunner.ExecuteTestruns(logger.Log.WithName("Execute"), config, run, testrunNamePrefix)
		if run.HasErrors() {
			logger.Log.Error(run.Errors(), "Testrunner execution disrupted")
			os.Exit(1)
		}

		if run[0].Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess {
			logger.Log.Info("Testrunner successfully finished.")
		} else {
			logger.Log.Error(errors.New("Testrunner finished unsuccessful"), "", "phase", run[0].Testrun.Status.Phase)
		}

		fmt.Print(util.PrettyPrintStruct(run[0].Testrun.Status))
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
	runTestrunCmd.Flags().Int64Var(&interval, "interval", 20, "Poll interval in seconds of the testrunner to poll for the testrun status.")

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
