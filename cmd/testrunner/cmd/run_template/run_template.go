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

package run_template

import (
	"fmt"
	"os"
	"time"

	"github.com/gardener/test-infra/pkg/logger"

	"github.com/gardener/test-infra/pkg/util"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	testrunnerTemplate "github.com/gardener/test-infra/pkg/testrunner/template"
	"github.com/spf13/cobra"
)

var testrunnerConfig = testrunner.Config{}
var collectConfig = result.Config{}
var shootParameters = testrunnerTemplate.Parameters{}

var (
	testrunNamePrefix    string
	shootPrefix          string
	tmKubeconfigPath     string
	filterPatchVersions  bool
	failOnError          bool
	testrunFlakeAttempts int

	timeout  int64
	interval int64
)

// AddCommand adds run-template to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run-template",
	Short: "Run the testrunner with a helm template containing testruns",
	Aliases: []string{
		"run", // for backward compatibility
		"run-tmpl",
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			err    error
			stopCh = make(chan struct{})
		)
		defer close(stopCh)
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		logger.Log.Info("Start testmachinery testrunner")

		testrunnerConfig.Watch, err = testrunner.StartWatchController(logger.Log, tmKubeconfigPath, stopCh)
		if err != nil {
			logger.Log.Error(err, "unable to start testrun watch controller")
			os.Exit(1)
		}

		gardenK8sClient, err := kubernetes.NewClientFromFile("", shootParameters.GardenKubeconfigPath, kubernetes.WithClientOptions(client.Options{
			Scheme: kubernetes.GardenScheme,
		}))
		if err != nil {
			logger.Log.Error(err, "unable to build garden kubernetes client", "file", tmKubeconfigPath)
			os.Exit(1)
		}

		testrunnerConfig.Timeout = time.Duration(timeout) * time.Second
		testrunnerConfig.FlakeAttempts = testrunFlakeAttempts
		collectConfig.ComponentDescriptorPath = shootParameters.ComponentDescriptorPath

		shootFlavors, err := GetShootFlavors(shootParameters.FlavorConfigPath, gardenK8sClient, shootPrefix, filterPatchVersions)
		if err != nil {
			logger.Log.Error(err, "unable to parse shoot flavors from test configuration")
			os.Exit(1)
		}

		runs, err := testrunnerTemplate.RenderTestruns(logger.Log.WithName("Render"), &shootParameters, shootFlavors.GetShoots())
		if err != nil {
			logger.Log.Error(err, "unable to render testrun")
			os.Exit(1)
		}

		if dryRun {
			fmt.Print(util.PrettyPrintStruct(runs))
			os.Exit(0)
		}

		collector, err := result.New(logger.Log.WithName("collector"), collectConfig, tmKubeconfigPath)
		if err != nil {
			logger.Log.Error(err, "unable to initialize collector")
			os.Exit(1)
		}
		if err := collector.PreRunShoots(shootParameters.GardenKubeconfigPath, runs); err != nil {
			logger.Log.Error(err, "unable to setup collector")
			os.Exit(1)
		}

		if err := testrunner.ExecuteTestruns(logger.Log.WithName("Execute"), &testrunnerConfig, runs, testrunNamePrefix, collector.RunExecCh); err != nil {
			logger.Log.Error(err, "unable to run testruns")
			os.Exit(1)
		}

		failed, err := collector.Collect(logger.Log.WithName("Collect"), testrunnerConfig.Watch.Client(), testrunnerConfig.Namespace, runs)
		if err != nil {
			logger.Log.Error(err, "unable to collect test output")
			os.Exit(1)
		}

		result.GenerateNotificationConfigForAlerting(runs.GetTestruns(), collectConfig.ConcourseOnErrorDir)

		logger.Log.Info("Testrunner finished")

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
		logger.Log.Error(err, "mark flag required", "flag", "tm-kubeconfig-path")
	}
	if err := runCmd.MarkFlagFilename("tm-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "tm-kubeconfig-path")
	}
	runCmd.Flags().StringVar(&testrunNamePrefix, "testrun-prefix", "default-", "Testrun name prefix which is used to generate a unique testrun name.")
	if err := runCmd.MarkFlagRequired("testrun-prefix"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "testrun-prefix")
	}
	runCmd.Flags().StringVarP(&testrunnerConfig.Namespace, "namespace", "n", "default", "Namesapce where the testrun should be deployed.")
	runCmd.Flags().Int64Var(&timeout, "timeout", 3600, "Timout in seconds of the testrunner to wait for the complete testrun to finish.")
	runCmd.Flags().Int64Var(&interval, "interval", 20, "Poll interval in seconds of the testrunner to poll for the testrun status.")
	runCmd.Flags().IntVar(&testrunFlakeAttempts, "testrun-flake-attempts", 0, "Max number of testruns until testrun is successful")
	runCmd.Flags().BoolVar(&failOnError, "fail-on-error", true, "Testrunners exits with 1 if one testruns failed.")
	runCmd.Flags().BoolVar(&collectConfig.EnableTelemetry, "enable-telemetry", false, "Enables the measurements of metrics during execution")
	runCmd.Flags().BoolVar(&testrunnerConfig.Serial, "serial", false, "executes all testruns of a bucket only after the previous bucket has finished")
	runCmd.Flags().IntVar(&testrunnerConfig.BackoffBucket, "backoff-bucket", 0, "Number of parallel created testruns per backoff period")
	runCmd.Flags().DurationVar(&testrunnerConfig.BackoffPeriod, "backoff-period", 0, "Time to wait between the creation of testrun buckets")

	runCmd.Flags().StringVar(&collectConfig.OutputDir, "output-dir-path", "./testout", "The filepath where the summary should be written to.")
	runCmd.Flags().StringVar(&collectConfig.ESConfigName, "es-config-name", "sap_internal", "The elasticsearch secret-server config name.")
	runCmd.Flags().StringVar(&collectConfig.S3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	runCmd.Flags().BoolVar(&collectConfig.S3SSL, "s3-ssl", false, "S3 has SSL enabled.")
	runCmd.Flags().StringVar(&collectConfig.ConcourseOnErrorDir, "concourse-onError-dir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")

	// status asset upload
	runCmd.Flags().BoolVar(&collectConfig.UploadStatusAsset, "upload-status-asset", false, "Upload testrun status as a github release asset.")
	runCmd.Flags().StringVar(&collectConfig.GithubUser, "github-user", os.Getenv("GITHUB_USER"), "On error dir which is used by Concourse.")
	runCmd.Flags().StringVar(&collectConfig.GithubPassword, "github-password", os.Getenv("GITHUB_PASSWORD"), "Github password.")
	runCmd.Flags().StringArrayVar(&collectConfig.AssetComponents, "asset-component", []string{}, "The github components to which the testrun status shall be attached as an asset.")
	runCmd.Flags().StringVar(&collectConfig.AssetPrefix, "asset-prefix", "", "Prefix of the asset name.")

	// parameter flags
	runCmd.Flags().StringVar(&shootParameters.DefaultTestrunChartPath, "testruns-chart-path", "", "Path to the default testruns chart.")
	if err := runCmd.MarkFlagFilename("testruns-chart-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "testruns-chart-path")
	}
	runCmd.Flags().StringVar(&shootParameters.FlavoredTestrunChartPath, "flavored-testruns-chart-path", "", "Path to the testruns chart to test shoots.")
	if err := runCmd.MarkFlagFilename("flavored-testruns-chart-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "flavored-testruns-chart-path")
	}
	runCmd.Flags().StringVar(&shootParameters.GardenKubeconfigPath, "gardener-kubeconfig-path", "", "Path to the gardener kubeconfig.")
	if err := runCmd.MarkFlagRequired("gardener-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "gardener-kubeconfig-path")
	}
	if err := runCmd.MarkFlagFilename("gardener-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "gardener-kubeconfig-path")
	}
	if err := runCmd.MarkFlagRequired("gardener-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "gardener-kubeconfig-path")
	}

	runCmd.Flags().StringVar(&shootParameters.FlavorConfigPath, "flavor-config", "", "Path to shoot test configuration.")
	if err := runCmd.MarkFlagFilename("flavor-config"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "flavor-config")
	}

	runCmd.Flags().StringVar(&shootPrefix, "shoot-name", "", "Shoot name which is used to run tests.")
	if err := runCmd.MarkFlagRequired("shoot-name"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "shoot-name")
	}
	runCmd.Flags().BoolVar(&filterPatchVersions, "filter-patch-versions", false, "Filters patch versions so that only the latest patch versions per minor versions is used.")

	runCmd.Flags().StringVar(&shootParameters.ComponentDescriptorPath, "component-descriptor-path", "", "Path to the component descriptor (BOM) of the current landscape.")
	runCmd.Flags().StringVar(&shootParameters.Landscape, "landscape", "", "Current gardener landscape.")

	runCmd.Flags().StringVar(&shootParameters.SetValues, "set", "", "setValues additional helm values")
	runCmd.Flags().StringArrayVarP(&shootParameters.FileValues, "values", "f", make([]string, 0), "yaml value files to override template values")
}
