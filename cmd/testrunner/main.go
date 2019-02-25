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

package main

import (
	"fmt"

	"github.com/gardener/test-infra/pkg/testrunner"

	"github.com/gardener/test-infra/pkg/util"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

var (
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
	componenetDescriptorPath string
	timeout                  int64

	outputFilePath          string
	elasticSearchConfigName string
	s3Endpoint              string
	concourseOnErrorDir     string
)

var rootCmd = &cobra.Command{
	Use:   "testrunner",
	Short: "Testrunner for Test Machinery",
	Run: func(cmd *cobra.Command, args []string) {

		cmd.DebugFlags()

		shootName = fmt.Sprintf("%s-%s", shootName, util.RandomString(5))
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
			ComponentDescriptorPath: componenetDescriptorPath,
		}

		testrunner.Run(config, parameters)
	},
}

func main() {
	log.Info("Start testmachinery testrunner")

	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err.Error())
	}
}
