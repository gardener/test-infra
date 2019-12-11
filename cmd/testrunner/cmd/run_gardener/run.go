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

package run_gardener

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/hostscheduler/gardenerscheduler"
	"github.com/gardener/test-infra/pkg/shootflavors"
	"github.com/gardener/test-infra/pkg/testmachinery/testrun"
	"github.com/gardener/test-infra/pkg/testrun_renderer"
	_default "github.com/gardener/test-infra/pkg/testrun_renderer/default"
	"github.com/gardener/test-infra/pkg/testrun_renderer/templates"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util/cmdvalues"
	"github.com/gardener/test-infra/pkg/util/gardensetup"
	"os"
	"time"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/util"

	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	"github.com/spf13/cobra"
)

var testrunnerConfig = testrunner.Config{}
var collectConfig = result.Config{}

var defaultConfig = _default.Config{}

var (
	tmKubeconfigPath string
	failOnError      bool

	testrunNamePrefix       string
	componentDescriptorPath string
	kubernetesVersions      []string
	cloudproviders          []common.CloudProvider
	gardenerExtensions      string
	testLabel               string
	hibernation             bool
)

// AddCommand adds run-gardener to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run-gardener",
	Short: "Run the testrunner with the default gardener test",
	Aliases: []string{
		"gardener",
	},
	Run: func(cmd *cobra.Command, args []string) {
		var (
			err    error
			stopCh = make(chan struct{})
		)
		defer close(stopCh)
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		logger.Log.Info("Start testmachinery testrunner")

		components, err := componentdescriptor.GetComponentsFromFile(componentDescriptorPath)
		if err != nil {
			logger.Log.Error(err, "unable to render default testrun")
			os.Exit(1)
		}

		rawFlavors := make([]*common.ShootFlavor, len(cloudproviders))
		for i, cp := range cloudproviders {
			versions := util.ConvertStringArrayToVersions(kubernetesVersions)
			rawFlavors[i] = &common.ShootFlavor{
				Provider: cp,
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &versions,
				},
			}
		}
		flavors, err := shootflavors.New(rawFlavors)
		if err != nil {
			logger.Log.Error(err, "unable to render default testrun")
			os.Exit(1)
		}

		defaultConfig.Shoots.Flavors = flavors
		defaultConfig.Components = components
		defaultConfig.Namespace = testrunnerConfig.Namespace
		defaultConfig.Shoots.DefaultTest = templates.TestWithLabels(testLabel)
		if hibernation {
			defaultConfig.Shoots.Tests = []testrun_renderer.TestsFunc{templates.HibernationLifecycle}
		}
		defaultConfig.GardenerExtensions, err = gardensetup.ParseFlag(gardenerExtensions)
		if err != nil {
			logger.Log.Error(err, "unable to parse gardener extensions")
			os.Exit(1)
		}

		tr, err := _default.Render(&defaultConfig)
		if err != nil {
			logger.Log.Error(err, "unable to render default testrun")
			os.Exit(1)
		}

		runs := testrunner.RunList{
			&testrunner.Run{
				Testrun:  tr,
				Metadata: nil,
				Error:    nil,
			},
		}

		if dryRun {
			fmt.Print(util.PrettyPrintStruct(tr))
			if err := testrun.Validate(logger.Log.WithName("validation"), tr); err != nil {
				fmt.Println(err.Error())
			}
			os.Exit(0)
		}

		testrunnerConfig.Watch, err = testrunner.StartWatchController(logger.Log, tmKubeconfigPath, stopCh)
		if err != nil {
			logger.Log.Error(err, "unable to start testrun watch controller")
			os.Exit(1)
		}

		testrunName := fmt.Sprintf("%s-", testrunNamePrefix)

		collector, err := result.New(logger.Log.WithName("collector"), collectConfig, tmKubeconfigPath)
		if err != nil {
			logger.Log.Error(err, "unable to initialize collector")
			os.Exit(1)
		}

		testrunner.ExecuteTestruns(logger.Log.WithName("Execute"), &testrunnerConfig, runs, testrunName)
		failed, err := collector.Collect(logger.Log.WithName("Collect"), testrunnerConfig.Watch.Client(), testrunnerConfig.Namespace, runs)
		if err != nil {
			logger.Log.Error(err, "unable to collect results")
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
	runCmd.Flags().StringVar(&tmKubeconfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to the testmachinery cluster kubeconfig")
	if err := runCmd.MarkFlagFilename("kubeconfig"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "kubeconfig")
	}
	runCmd.Flags().StringVar(&testrunNamePrefix, "testrun-prefix", "default-", "Testrun name prefix which is used to generate a unique testrun name.")
	runCmd.Flags().StringVarP(&testrunnerConfig.Namespace, "namespace", "n", "default", "Namespace where the testrun should be deployed.")
	runCmd.Flags().Var(cmdvalues.NewDurationValue(&testrunnerConfig.Timeout, time.Hour), "timeout", "Timout the testrunner to wait for the complete testrun to finish. Valid time units are 'ns', 'us' (or 'µs'), 'ms', 's', 'm', 'h'.")
	runCmd.Flags().String("interval", "20s", "[DEPRECTAED] Value has no effect on the testrunner")
	runCmd.Flags().BoolVar(&failOnError, "fail-on-error", true, "Testrunners exits with 1 if one testruns failed.")

	runCmd.Flags().StringVar(&collectConfig.OutputDir, "output-dir-path", "./testout", "The filepath where the summary should be written to.")
	runCmd.Flags().StringVar(&collectConfig.ESConfigName, "es-config-name", "sap_internal", "The elasticsearch secret-server config name.")
	runCmd.Flags().StringVar(&collectConfig.S3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	runCmd.Flags().BoolVar(&collectConfig.S3SSL, "s3-ssl", false, "S3 has SSL enabled.")
	runCmd.Flags().StringVar(&collectConfig.ConcourseOnErrorDir, "concourse-onError-dir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")

	runCmd.Flags().StringVar(&componentDescriptorPath, "component-descriptor-path", "", "Path to the component descriptor (BOM) of the current landscape.")

	runCmd.Flags().Var(cmdvalues.NewHostProviderValue(&defaultConfig.HostProvider, gardenerscheduler.Name), "hostprovider", "Specify the provider for selecting the base cluster")
	runCmd.Flags().StringVar(&defaultConfig.GardenSetupRevision, "garden-setup-version", "master", "Specify the garden setup version to setup gardener")
	runCmd.Flags().Var(cmdvalues.NewCloudProviderValue(&defaultConfig.BaseClusterCloudprovider, common.CloudProviderGCP, common.CloudProviderGCP, common.CloudProviderAWS, common.CloudProviderAzure),
		"host-cloudprovider", "Specify the cloudprovider of the host cluster. Optional and only affect gardener base cluster")
	runCmd.Flags().StringVar(&defaultConfig.Gardener.Version, "gardener-version", "", "Specify the gardener version to be deployed by garden setup")
	runCmd.Flags().StringVar(&defaultConfig.Gardener.ImageTag, "gardener-image", "", "Specify the gardener image tag to be deployed by garden setup")
	runCmd.Flags().StringVar(&defaultConfig.Gardener.Commit, "gardener-commit", "", "Specify the gardener commit that is deployed by garden setup")
	runCmd.Flags().StringVar(&gardenerExtensions, "gardener-extensions", "provider-gcp=github.com/gardener/gardener-extensions.git::master", "Specify the gardener extensions versions to be deployed by garden setup")

	runCmd.Flags().StringVar(&defaultConfig.Shoots.Namespace, "project-namespace", "garden-core", "Specify the shoot namespace where the shoots should be created")
	runCmd.Flags().StringArrayVar(&kubernetesVersions, "kubernetes-version", []string{}, "Specify the kubernetes version to test")
	runCmd.Flags().VarP(cmdvalues.NewCloudProviderArrayValue(&cloudproviders, common.CloudProviderGCP, common.CloudProviderAWS, common.CloudProviderAzure), "cloudprovider", "p", "Specify the cloudproviders to test.")

	runCmd.Flags().StringVarP(&testLabel, "label", "l", string(testmachinery.TestLabelDefault), "Specify test label that should be fetched by the testmachinery")
	runCmd.Flags().BoolVar(&hibernation, "hibernation", false, "test hibernation")

}
