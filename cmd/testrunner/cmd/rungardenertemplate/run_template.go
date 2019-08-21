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

package rungardenertemplate

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/util"
	"os"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	testrunnerTemplate "github.com/gardener/test-infra/pkg/testrunner/template"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

var (
	tmKubeconfigPath string
	namespace        string
	timeout          int64
	interval         int64
	failOnError      bool

	outputDirPath           string
	elasticSearchConfigName string
	s3Endpoint              string
	s3SSL                   bool
	concourseOnErrorDir     string

	testrunChartPath                string
	testrunNamePrefix               string
	componentDescriptorPath         string
	upgradedComponentDescriptorPath string

	// optional
	landscape                string
	gardenerCurrentVersion   string
	gardenerCurrentRevision  string
	gardenerUpgradedVersion  string
	gardenerUpgradedRevision string
	setValues                string
	fileValues               []string
)

// AddCommand adds run-testrun to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run-gardener-template",
	Short: "Run the testrunner with a helm template containing testruns",
	Aliases: []string{
		"run-full",
		"run-gardener",
		"run-tmpl-full",
	},
	Run: func(cmd *cobra.Command, args []string) {
		debug, _ := cmd.Flags().GetBool("debug")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if debug {
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

		testrunName := fmt.Sprintf("%s-", testrunNamePrefix)
		config := &testrunner.Config{
			TmClient:  tmClient,
			Namespace: namespace,
			Timeout:   timeout,
			Interval:  interval,
		}

		rsConfig := &result.Config{
			OutputDir:           outputDirPath,
			ESConfigName:        elasticSearchConfigName,
			S3Endpoint:          s3Endpoint,
			S3SSL:               s3SSL,
			ConcourseOnErrorDir: concourseOnErrorDir,
		}

		parameters := &testrunnerTemplate.GardenerTestrunParameters{
			TestrunChartPath: testrunChartPath,
			Namespace:        namespace,

			ComponentDescriptorPath:         componentDescriptorPath,
			UpgradedComponentDescriptorPath: upgradedComponentDescriptorPath,
			SetValues:                       setValues,
			FileValues:                      fileValues,
		}

		metadata := &testrunner.Metadata{
			Landscape: landscape,
		}
		runs, err := testrunnerTemplate.RenderGardenerTestrun(tmClient, parameters, metadata)
		if err != nil {
			log.Fatal(err)
		}

		if dryRun {
			fmt.Print(util.PrettyPrintStruct(runs))
			os.Exit(0)
		}

		testrunner.ExecuteTestruns(config, runs, testrunName)
		failed, err := result.Collect(rsConfig, tmClient, config.Namespace, runs)
		if err != nil {
			log.Fatal(err)
		}

		result.GenerateNotificationConfigForAlerting(runs.GetTestruns(), rsConfig.ConcourseOnErrorDir)

		log.Info("Testrunner finished.")
		// Fail when one testrun is failed and we should fail on failed testruns.
		// Otherwise only fail when the testrun execution is erroneous.
		if runs.HasErrors() {
			os.Exit(1)
		}
		if failOnError && failed {
			os.Exit(1)
		}
	},
}

func init() {
	// configuration flags
	runCmd.Flags().StringVar(&tmKubeconfigPath, "tm-kubeconfig-path", "", "Path to the testmachinery cluster kubeconfig")
	if err := runCmd.MarkFlagRequired("tm-kubeconfig-path"); err != nil {
		log.Debug(err.Error())
	}
	if err := runCmd.MarkFlagFilename("tm-kubeconfig-path"); err != nil {
		log.Debug(err.Error())
	}
	runCmd.Flags().StringVar(&testrunNamePrefix, "testrun-prefix", "default-", "Testrun name prefix which is used to generate a unique testrun name.")
	runCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namesapce where the testrun should be deployed.")
	runCmd.Flags().Int64Var(&timeout, "timeout", 3600, "Timout in seconds of the testrunner to wait for the complete testrun to finish.")
	runCmd.Flags().Int64Var(&interval, "interval", 20, "Poll interval in seconds of the testrunner to poll for the testrun status.")
	runCmd.Flags().BoolVar(&failOnError, "fail-on-error", true, "Testrunners exits with 1 if one testruns failed.")

	runCmd.Flags().StringVar(&outputDirPath, "output-dir-path", "./testout", "The filepath where the summary should be written to.")
	runCmd.Flags().StringVar(&elasticSearchConfigName, "es-config-name", "sap_internal", "The elasticsearch secret-server config name.")
	runCmd.Flags().StringVar(&s3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	runCmd.Flags().BoolVar(&s3SSL, "s3-ssl", false, "S3 has SSL enabled.")
	runCmd.Flags().StringVar(&concourseOnErrorDir, "concourse-onError-dir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")

	// parameter flags
	runCmd.Flags().StringVar(&testrunChartPath, "testruns-chart-path", "", "Path to the testruns chart.")
	if err := runCmd.MarkFlagRequired("testruns-chart-path"); err != nil {
		log.Debug(err.Error())
	}
	if err := runCmd.MarkFlagFilename("testruns-chart-path"); err != nil {
		log.Debug(err.Error())
	}

	runCmd.Flags().StringVar(&componentDescriptorPath, "component-descriptor-path", "", "Path to the component descriptor (BOM) of the current landscape.")
	runCmd.Flags().StringVar(&upgradedComponentDescriptorPath, "upgraded-component-descriptor-path", "", "Path to the component descriptor (BOM) of the new landscape.")

	runCmd.Flags().StringVar(&landscape, "landscape", "", "Current gardener landscape.")
	runCmd.Flags().StringVar(&gardenerCurrentVersion, "gardener-current-version", "", "Set current version of gardener. This will result in the helm value {{ .Values.gardener.current.version }}")
	runCmd.Flags().StringVar(&gardenerCurrentRevision, "gardener-current-revision", "", "Set current revision of gardener. This will result in the helm value {{ .Values.gardener.current.revision }}")
	runCmd.Flags().StringVar(&gardenerUpgradedVersion, "gardener-upgraded-version", "", "Set current version of gardener. This will result in the helm value {{ .Values.gardener.upgraded.version }}")
	runCmd.Flags().StringVar(&gardenerUpgradedRevision, "gardener-upgraded-revision", "", "Set current revision of gardener. This will result in the helm value {{ .Values.gardener.upgraded.revision }}")

	runCmd.Flags().StringVar(&setValues, "set", "", "setValues additional helm values")
	runCmd.Flags().StringArrayVarP(&fileValues, "values", "f", make([]string, 0), "yaml value files to override template values")
}
