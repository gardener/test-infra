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

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/util"
	"github.com/joho/godotenv"

	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"

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
		if debug, _ := cmd.Flags().GetBool("debug"); debug {
			log.SetLevel(log.DebugLevel)
			log.Warn("Set debug log level")

			cmd.DebugFlags()
		}
		log.Info("Start testmachinery testrunner")
		err := godotenv.Load()
		if err == nil {
			log.Debug(".env file loaded")
		} else {
			log.Debugf("Error loading .env file: %s", err.Error())
		}

		tmClient, err := kubernetes.NewClientFromFile("", tmKubeconfigPath, client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		})
		if err != nil {
			log.Fatalf("Cannot build kubernetes client from %s: %s", tmKubeconfigPath, err.Error())
		}

		config := &testrunner.Config{
			TmClient:  tmClient,
			Namespace: namespace,
			Timeout:   timeout,
			Interval:  interval,
		}

		tr, err := util.ParseTestrunFromFile(testrunPath)
		if err != nil {
			log.Fatalf("Testrunner execution disrupted: %s", err.Error())
		}

		finishedTestruns, err := testrunner.Run(config, []*tmv1beta1.Testrun{&tr}, testrunNamePrefix)
		if err != nil {
			log.Fatalf("Testrunner execution disrupted: %s", err.Error())
		}

		if finishedTestruns[0].Status.Phase == tmv1beta1.PhaseStatusSuccess {
			log.Info("Testrunner successfully finished.")
		} else {
			log.Errorf("Testrunner finished with phase %s", finishedTestruns[0].Status.Phase)
		}

		fmt.Print(util.PrettyPrintStruct(finishedTestruns[0].Status))
	},
}

func init() {
	// configuration flags
	runTestrunCmd.Flags().StringVar(&tmKubeconfigPath, "tm-kubeconfig-path", "", "Path to the testmachinery cluster kubeconfig")
	runTestrunCmd.MarkFlagRequired("tm-kubeconfig-path")
	runTestrunCmd.MarkFlagFilename("tm-kubeconfig-path")
	runTestrunCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace where the testrun should be deployed.")

	runTestrunCmd.Flags().Int64Var(&timeout, "timeout", 3600, "Timout in seconds of the testrunner to wait for the complete testrun to finish.")
	runTestrunCmd.Flags().Int64Var(&interval, "interval", 20, "Poll interval in seconds of the testrunner to poll for the testrun status.")

	// parameter flags
	runTestrunCmd.Flags().StringVarP(&testrunPath, "file", "f", "", "Path to the testrun yaml")
	runTestrunCmd.MarkFlagRequired("gardener-kubeconfig-path")
	runTestrunCmd.MarkFlagFilename("tm-kubeconfig-path")
	runTestrunCmd.Flags().StringVar(&testrunNamePrefix, "name-prefix", "testrunner-", "Name prefix of the testrun")

}
