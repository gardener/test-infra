package cmd

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

import (
	"fmt"
	"os"

	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

var (
	debug bool

	gardenKubeconfigPath     string
	tmKubeconfigPath         string
	testrunChartPath         string
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
	autoscalerMin            string
	autoscalerMax            string
	floatingPoolName         string
	componenetDescriptorPath string
	timeout                  int64

	outputFilePath          string
	elasticSearchConfigName string
	s3Endpoint              string
	concourseOnErrorDir     string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the tesrunner with a testrun helm template",
	Run: func(cmd *cobra.Command, args []string) {

		if debug {
			log.SetLevel(log.DebugLevel)
			log.Warn("Set debug log level")

			cmd.DebugFlags()
		}

		testrunName := fmt.Sprintf("%s-%s-", testrunNamePrefix, cloudprovider)

		config := &testrunner.TestrunConfig{
			TmKubeconfigPath:    tmKubeconfigPath,
			Timeout:             &timeout,
			OutputFile:          outputFilePath,
			ESConfigName:        elasticSearchConfigName,
			S3Endpoint:          s3Endpoint,
			ConcourseOnErrorDir: concourseOnErrorDir,
		}

		parameters := &testrunner.TestrunParameters{
			GardenKubeconfigPath: gardenKubeconfigPath,
			TestrunName:          testrunName,
			TestrunChartPath:     testrunChartPath,

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
			AutoscalerMin:           autoscalerMin,
			AutoscalerMax:           autoscalerMax,
			FloatingPoolName:        floatingPoolName,
			ComponentDescriptorPath: componenetDescriptorPath,
		}

		testrunner.Run(config, parameters)
	},
}

func init() {
	err := godotenv.Load()
	if err == nil {
		log.Info(".env file loaded")
	} else {
		log.Warnf("Error loading .env file: %s", err.Error())
	}

	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	log.SetOutput(os.Stderr)

	// Set commandline flags

	// configuration flags
	runCmd.Flags().StringVar(&tmKubeconfigPath, "tm-kubeconfig-path", "", "Path to the testmachinery cluster kubeconfig")
	runCmd.MarkFlagRequired("tm-kubeconfig-path")
	runCmd.MarkFlagFilename("tm-kubeconfig-path")
	runCmd.Flags().StringVar(&testrunChartPath, "testruns-chart-path", "", "Path to the testruns chart.")
	runCmd.MarkFlagRequired("testruns-chart-path")
	runCmd.MarkFlagFilename("testruns-chart-path")
	runCmd.Flags().StringVar(&testrunNamePrefix, "testrun-prefix", "default-", "Testrun name prefix which is used to generate a unique testrun name.")
	runCmd.MarkFlagRequired("testrun-prefix")

	runCmd.Flags().Int64Var(&timeout, "timeout", -1, "timout of the testrunner to wait for the complete testrun to finish.")
	runCmd.Flags().StringVar(&outputFilePath, "output-file-path", "./testout", "The filepath where the summary should be written to.")
	runCmd.Flags().StringVar(&elasticSearchConfigName, "es-config-name", "sap_internal", "The elasticsearch secret-server config name.")
	runCmd.Flags().StringVar(&s3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	runCmd.Flags().StringVar(&concourseOnErrorDir, "concourse-onError-dir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")

	// parameter flags
	runCmd.Flags().StringVar(&gardenKubeconfigPath, "gardener-kubeconfig-path", "", "Path to the gardener kubeconfig.")
	runCmd.MarkFlagRequired("gardener-kubeconfig-path")
	runCmd.MarkFlagFilename("gardener-kubeconfig-path")
	runCmd.Flags().StringVar(&projectName, "project-name", "", "Gardener project name of the shoot")
	runCmd.MarkFlagRequired("gardener-kubeconfig-path")
	runCmd.Flags().StringVar(&shootName, "shoot-name", "", "Shoot name which is used to run tests.")
	runCmd.MarkFlagRequired("gardener-kubeconfig-path")
	runCmd.Flags().StringVar(&cloudprovider, "cloudprovider", "", "Cloudprovider where the shoot is created.")
	runCmd.MarkFlagRequired("gardener-kubeconfig-path")
	runCmd.Flags().StringVar(&cloudprofile, "cloudprofile", "", "Cloudprofile of shoot.")
	runCmd.MarkFlagRequired("gardener-kubeconfig-path")
	runCmd.Flags().StringVar(&secretBinding, "secret-binding", "", "SecretBinding that should be used to create the shoot.")
	runCmd.MarkFlagRequired("gardener-kubeconfig-path")
	runCmd.Flags().StringVar(&region, "region", "", "Region where the shoot is created.")
	runCmd.MarkFlagRequired("gardener-kubeconfig-path")
	runCmd.Flags().StringVar(&zone, "zone", "", "Zone of the shoot worker nodes. Not required for azure shoots.")
	runCmd.MarkFlagRequired("gardener-kubeconfig-path")

	runCmd.Flags().StringVar(&k8sVersion, "k8s-version", "", "Kubernetes version of the shoot.")
	runCmd.Flags().StringVar(&machineType, "machinetype", "", "Machinetype of the shoot's worker nodes.")
	runCmd.Flags().StringVar(&autoscalerMin, "autoscaler-min", "", "Min number of worker nodes.")
	runCmd.Flags().StringVar(&autoscalerMax, "autoscaler-max", "", "Max number of worker nodes.")
	runCmd.Flags().StringVar(&floatingPoolName, "floating-pool-name", "", "Floating pool name where the cluster is created. Only needed for Openstack.")
	runCmd.Flags().StringVar(&componenetDescriptorPath, "component-descriptor-path", "", "Path to the component descriptor (BOM) of the current landscape.")
	runCmd.Flags().StringVar(&landscape, "landscape", "", "Current gardener landscape.")

	rootCmd.AddCommand(runCmd)
}
