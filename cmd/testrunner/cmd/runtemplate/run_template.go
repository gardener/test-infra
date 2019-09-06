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

package runtemplate

import (
	"fmt"
	"os"

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
	argouiEndpoint          string
	concourseOnErrorDir     string

	testrunChartPath         string
	gardenKubeconfigPath     string
	allK8sVersions           bool
	testrunNamePrefix        string
	projectName              string
	shootName                string
	landscape                string
	cloudprovider            string
	cloudprofile             string
	secretBinding            string
	region                   string
	zone                     string
	k8sVersion               string
	machineType              string
	machineImage             string
	machineImageVersion      string
	autoscalerMin            string
	autoscalerMax            string
	floatingPoolName         string
	loadbalancerProvider     string
	componenetDescriptorPath string

	setValues  string
	fileValues []string
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
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		logger.Log.Info("Start testmachinery testrunner")

		tmClient, err := kubernetes.NewClientFromFile("", tmKubeconfigPath, kubernetes.WithClientOptions(client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		}))
		if err != nil {
			logger.Log.Error(err, "unable to build kubernetes client", "file", tmKubeconfigPath)
			os.Exit(1)
		}

		testrunComputedPrefix := fmt.Sprintf("%s-%s-", testrunNamePrefix, cloudprovider)
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
			ArgoUIEndpoint:      argouiEndpoint,
			ConcourseOnErrorDir: concourseOnErrorDir,
		}

		parameters := &testrunnerTemplate.ShootTestrunParameters{
			GardenKubeconfigPath: gardenKubeconfigPath,
			TestrunChartPath:     testrunChartPath,
			MakeVersionMatrix:    allK8sVersions,

			ProjectName:             projectName,
			ShootName:               shootName,
			Landscape:               landscape,
			Cloudprovider:           cloudprovider,
			Cloudprofile:            cloudprofile,
			SecretBinding:           secretBinding,
			Region:                  region,
			Zone:                    zone,
			K8sVersion:              k8sVersion,
			MachineType:             machineType,
			MachineImage:            machineImage,
			MachineImageVersion:     machineImageVersion,
			AutoscalerMin:           autoscalerMin,
			AutoscalerMax:           autoscalerMax,
			FloatingPoolName:        floatingPoolName,
			LoadBalancerProvider:    loadbalancerProvider,
			ComponentDescriptorPath: componenetDescriptorPath,
			SetValues:               setValues,
		}

		metadata := &testrunner.Metadata{
			Landscape:         parameters.Landscape,
			CloudProvider:     parameters.Cloudprovider,
			KubernetesVersion: parameters.K8sVersion,
		}
		runs, err := testrunnerTemplate.RenderShootTestrun(logger.Log.WithName("Render"), tmClient, parameters, metadata)
		if err != nil {
			logger.Log.Error(err, "unable to render testrun")
			os.Exit(1)
		}

		if dryRun {
			fmt.Print(util.PrettyPrintStruct(runs))
			os.Exit(0)
		}

		testrunner.ExecuteTestruns(logger.Log.WithName("Execute"), config, runs, testrunComputedPrefix)

		failed, err := result.Collect(logger.Log.WithName("Collect"), rsConfig, tmClient, config.Namespace, runs)
		if err != nil {
			logger.Log.Error(err, "unable to collect test output")
			os.Exit(1)
		}

		result.GenerateNotificationConfigForAlerting(runs.GetTestruns(), rsConfig.ConcourseOnErrorDir)

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
	runCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namesapce where the testrun should be deployed.")
	runCmd.Flags().Int64Var(&timeout, "timeout", 3600, "Timout in seconds of the testrunner to wait for the complete testrun to finish.")
	runCmd.Flags().Int64Var(&interval, "interval", 20, "Poll interval in seconds of the testrunner to poll for the testrun status.")
	runCmd.Flags().BoolVar(&failOnError, "fail-on-error", true, "Testrunners exits with 1 if one testruns failed.")

	runCmd.Flags().StringVar(&outputDirPath, "output-dir-path", "./testout", "The filepath where the summary should be written to.")
	runCmd.Flags().StringVar(&elasticSearchConfigName, "es-config-name", "sap_internal", "The elasticsearch secret-server config name.")
	runCmd.Flags().StringVar(&s3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	runCmd.Flags().BoolVar(&s3SSL, "s3-ssl", false, "S3 has SSL enabled.")
	runCmd.Flags().StringVar(&argouiEndpoint, "argoui-endpoint", "", "ArgoUI endpoint of the testmachinery cluster.")
	runCmd.Flags().StringVar(&concourseOnErrorDir, "concourse-onError-dir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")

	// parameter flags
	runCmd.Flags().StringVar(&testrunChartPath, "testruns-chart-path", "", "Path to the testruns chart.")
	if err := runCmd.MarkFlagRequired("testruns-chart-path"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "testruns-chart-path")
	}
	if err := runCmd.MarkFlagFilename("testruns-chart-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "testruns-chart-path")
	}
	runCmd.Flags().StringVar(&gardenKubeconfigPath, "gardener-kubeconfig-path", "", "Path to the gardener kubeconfig.")
	if err := runCmd.MarkFlagRequired("gardener-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "gardener-kubeconfig-path")
	}
	if err := runCmd.MarkFlagFilename("gardener-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "gardener-kubeconfig-path")
	}
	runCmd.Flags().BoolVar(&allK8sVersions, "all-k8s-versions", false, "Run the testrun with all available versions specified by the cloudprovider.")
	runCmd.Flags().StringVar(&projectName, "project-name", "", "Gardener project name of the shoot")
	if err := runCmd.MarkFlagRequired("gardener-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "gardener-kubeconfig-path")
	}
	runCmd.Flags().StringVar(&shootName, "shoot-name", "", "Shoot name which is used to run tests.")
	if err := runCmd.MarkFlagRequired("shoot-name"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "shoot-name")
	}
	runCmd.Flags().StringVar(&cloudprovider, "cloudprovider", "", "Cloudprovider where the shoot is created.")
	if err := runCmd.MarkFlagRequired("cloudprovider"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "cloudprovider")
	}
	runCmd.Flags().StringVar(&cloudprofile, "cloudprofile", "", "Cloudprofile of shoot.")
	if err := runCmd.MarkFlagRequired("cloudprofile"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "cloudprofile")
	}
	runCmd.Flags().StringVar(&secretBinding, "secret-binding", "", "SecretBinding that should be used to create the shoot.")
	if err := runCmd.MarkFlagRequired("secret-binding"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "secret-binding")
	}
	runCmd.Flags().StringVar(&region, "region", "", "Region where the shoot is created.")
	if err := runCmd.MarkFlagRequired("region"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "region")
	}

	runCmd.Flags().StringVar(&zone, "zone", "", "Zone of the shoot worker nodes. Not required for azure shoots.")
	runCmd.Flags().StringVar(&k8sVersion, "k8s-version", "", "Kubernetes version of the shoot.")
	runCmd.Flags().StringVar(&machineType, "machinetype", "", "Machinetype of the shoot's worker nodes.")
	runCmd.Flags().StringVar(&machineImage, "machine-image", "", "Image of the OS running on the machine")
	runCmd.Flags().StringVar(&machineImageVersion, "machine-image-version", "", "The version of the machine image")
	runCmd.Flags().StringVar(&autoscalerMin, "autoscaler-min", "", "Min number of worker nodes.")
	runCmd.Flags().StringVar(&autoscalerMax, "autoscaler-max", "", "Max number of worker nodes.")
	runCmd.Flags().StringVar(&floatingPoolName, "floating-pool-name", "", "Floating pool name where the cluster is created. Only needed for Openstack.")
	runCmd.Flags().StringVar(&loadbalancerProvider, "loadbalancer-provider", "", "LoadBalancer Provider like haproxy. Only applicable for Openstack.")
	runCmd.Flags().StringVar(&componenetDescriptorPath, "component-descriptor-path", "", "Path to the component descriptor (BOM) of the current landscape.")
	runCmd.Flags().StringVar(&landscape, "landscape", "", "Current gardener landscape.")

	runCmd.Flags().StringVar(&setValues, "set", "", "setValues additional helm values")
	runCmd.Flags().StringArrayVarP(&fileValues, "values", "f", make([]string, 0), "yaml value files to override template values")
}
