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
	"github.com/gardener/test-infra/pkg/testmachinery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	testrunnerTemplate "github.com/gardener/test-infra/pkg/testrunner/template"
	"github.com/spf13/cobra"
)

var testrunnerConfig = testrunner.Config{}
var collectConfig = result.Config{}
var shootParameters = testrunnerTemplate.ShootTestrunParameters{}

var (
	testrunNamePrefix string
	tmKubeconfigPath  string
	failOnError       bool

	timeout  int64
	interval int64
)

// AddCommand adds run-testrun to a command.
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
		var err error
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		logger.Log.Info("Start testmachinery testrunner")

		testrunnerConfig.Client, err = kubernetes.NewClientFromFile("", tmKubeconfigPath, kubernetes.WithClientOptions(client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		}))
		if err != nil {
			logger.Log.Error(err, "unable to build kubernetes client", "file", tmKubeconfigPath)
			os.Exit(1)
		}

		testrunnerConfig.Timeout = time.Duration(timeout) * time.Second
		testrunnerConfig.Interval = time.Duration(interval) * time.Second

		testrunComputedPrefix := fmt.Sprintf("%s-%s-", testrunNamePrefix, shootParameters.Cloudprovider)
		metadata := &testrunner.Metadata{
			Landscape:         shootParameters.Landscape,
			CloudProvider:     shootParameters.Cloudprovider,
			KubernetesVersion: shootParameters.K8sVersion,
		}
		shootRuns, err := testrunnerTemplate.RenderShootTestruns(logger.Log.WithName("Render"), testrunnerConfig.Client, &shootParameters, metadata)
		if err != nil {
			logger.Log.Error(err, "unable to render testrun")
			os.Exit(1)
		}
		runs := shootRuns.Runs()

		if dryRun {
			fmt.Print(util.PrettyPrintStruct(runs))
			os.Exit(0)
		}

		collector, err := result.New(logger.Log.WithName("collector"), collectConfig)
		if err != nil {
			logger.Log.Error(err, "unable to initialize collector")
			os.Exit(1)
		}
		if err := collector.PreRunShoots(shootParameters.GardenKubeconfigPath, shootRuns); err != nil {
			logger.Log.Error(err, "unable to setup collector")
			os.Exit(1)
		}

		testrunner.ExecuteTestruns(logger.Log.WithName("Execute"), &testrunnerConfig, runs, testrunComputedPrefix)

		failed, err := collector.Collect(logger.Log.WithName("Collect"), testrunnerConfig.Client, testrunnerConfig.Namespace, runs)
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
	runCmd.Flags().BoolVar(&failOnError, "fail-on-error", true, "Testrunners exits with 1 if one testruns failed.")
	runCmd.Flags().BoolVar(&collectConfig.EnableTelemetry, "enable-telemetry", false, "Enables the measurements of metrics during execution")

	runCmd.Flags().StringVar(&collectConfig.OutputDir, "output-dir-path", "./testout", "The filepath where the summary should be written to.")
	runCmd.Flags().StringVar(&collectConfig.ESConfigName, "es-config-name", "sap_internal", "The elasticsearch secret-server config name.")
	runCmd.Flags().StringVar(&collectConfig.S3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	runCmd.Flags().BoolVar(&collectConfig.S3SSL, "s3-ssl", false, "S3 has SSL enabled.")
	runCmd.Flags().StringVar(&collectConfig.ConcourseOnErrorDir, "concourse-onError-dir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")
	runCmd.Flags().StringVar(&collectConfig.GithubUser, "github-user", os.Getenv("GITHUB_USER"), "On error dir which is used by Concourse.")
	runCmd.Flags().StringVar(&collectConfig.GithubPassword, "github-password", os.Getenv("GITHUB_PASSWORD"), "Github password.")
	runCmd.Flags().StringVar(&collectConfig.AssetComponent, "asset-component", "", "The github component to which the testrun status shall be attached as an asset.")
	runCmd.Flags().BoolVar(&collectConfig.UploadStatusAsset, "upload-status-asset", false, "Upload testrun status as a github release asset.")

	// parameter flags
	runCmd.Flags().StringVar(&shootParameters.TestrunChartPath, "testruns-chart-path", "", "Path to the testruns chart.")
	if err := runCmd.MarkFlagRequired("testruns-chart-path"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "testruns-chart-path")
	}
	if err := runCmd.MarkFlagFilename("testruns-chart-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "testruns-chart-path")
	}
	runCmd.Flags().StringVar(&shootParameters.GardenKubeconfigPath, "gardener-kubeconfig-path", "", "Path to the gardener kubeconfig.")
	if err := runCmd.MarkFlagRequired("gardener-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "gardener-kubeconfig-path")
	}
	if err := runCmd.MarkFlagFilename("gardener-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "gardener-kubeconfig-path")
	}
	runCmd.Flags().BoolVar(&shootParameters.MakeVersionMatrix, "all-k8s-versions", false, "Run the testrun with all available versions specified by the cloudprovider.")
	runCmd.Flags().StringVar(&shootParameters.ProjectName, "project-name", "", "Gardener project name of the shoot")
	if err := runCmd.MarkFlagRequired("gardener-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "gardener-kubeconfig-path")
	}
	runCmd.Flags().StringVar(&shootParameters.ShootName, "shoot-name", "", "Shoot name which is used to run tests.")
	if err := runCmd.MarkFlagRequired("shoot-name"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "shoot-name")
	}
	runCmd.Flags().StringVar(&shootParameters.Cloudprovider, "cloudprovider", "", "Cloudprovider where the shoot is created.")
	if err := runCmd.MarkFlagRequired("cloudprovider"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "cloudprovider")
	}
	runCmd.Flags().StringVar(&shootParameters.Cloudprofile, "cloudprofile", "", "Cloudprofile of shoot.")
	if err := runCmd.MarkFlagRequired("cloudprofile"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "cloudprofile")
	}
	runCmd.Flags().StringVar(&shootParameters.SecretBinding, "secret-binding", "", "SecretBinding that should be used to create the shoot.")
	if err := runCmd.MarkFlagRequired("secret-binding"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "secret-binding")
	}
	runCmd.Flags().StringVar(&shootParameters.Region, "region", "", "Region where the shoot is created.")
	if err := runCmd.MarkFlagRequired("region"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "region")
	}

	runCmd.Flags().StringVar(&shootParameters.Zone, "zone", "", "Zone of the shoot worker nodes. Not required for azure shoots.")
	runCmd.Flags().StringVar(&shootParameters.K8sVersion, "k8s-version", "", "Kubernetes version of the shoot.")
	runCmd.Flags().StringVar(&shootParameters.MachineType, "machinetype", "", "Machinetype of the shoot's worker nodes.")
	runCmd.Flags().StringVar(&shootParameters.MachineImage, "machine-image", "", "Image of the OS running on the machine")
	runCmd.Flags().StringVar(&shootParameters.MachineImageVersion, "machine-image-version", "", "The version of the machine image")
	runCmd.Flags().StringVar(&shootParameters.AutoscalerMin, "autoscaler-min", "", "Min number of worker nodes.")
	runCmd.Flags().StringVar(&shootParameters.AutoscalerMax, "autoscaler-max", "", "Max number of worker nodes.")
	runCmd.Flags().StringVar(&shootParameters.FloatingPoolName, "floating-pool-name", "", "Floating pool name where the cluster is created. Only needed for Openstack.")
	runCmd.Flags().StringVar(&shootParameters.LoadBalancerProvider, "loadbalancer-provider", "", "LoadBalancer Provider like haproxy. Only applicable for Openstack.")
	runCmd.Flags().StringVar(&shootParameters.ComponentDescriptorPath, "component-descriptor-path", "", "Path to the component descriptor (BOM) of the current landscape.")
	runCmd.Flags().StringVar(&shootParameters.Landscape, "landscape", "", "Current gardener landscape.")

	runCmd.Flags().StringVar(&shootParameters.SetValues, "set", "", "setValues additional helm values")
	runCmd.Flags().StringArrayVarP(&shootParameters.FileValues, "values", "f", make([]string, 0), "yaml value files to override template values")
}
