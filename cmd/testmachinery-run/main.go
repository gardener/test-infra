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
	"flag"
	"fmt"
	"os"

	"github.com/gardener/test-infra/cmd/testmachinery-run/testrunner"

	"github.com/gardener/test-infra/pkg/util"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

var (
	gardenKubeconfigPath string
	tmKubeconfigPath     string
	testrunChartPath     string
	projectName          string
	shootName            string
	landscape            string
	cloudprovider        string
	cloudprofile         string
	secretBinding        string
	region               string
	zone                 string
	k8sVersion           string
	bom                  string
	timeout              int64

	outputFilePath          string
	elasticSearchConfigName string
	s3Endpoint              string
)

func main() {
	log.Info("Start testmachinery testrunner")

	flag.Parse()

	shootName = fmt.Sprintf("%s-%s-%s", shootName, cloudprovider, util.RandomString(5))
	testrunName := fmt.Sprintf("demorun-%s", util.RandomString(5))

	config := &testrunner.TestrunConfig{
		TmKubeconfigPath:     tmKubeconfigPath,
		GardenKubeconfigPath: gardenKubeconfigPath,
		Timeout:              &timeout,
		OutputFile:           outputFilePath,
		ESConfigName:         elasticSearchConfigName,
		S3Endpoint:           s3Endpoint,
	}

	parameters := &testrunner.TestrunParameters{
		TestrunName:      testrunName,
		TestrunChartPath: testrunChartPath,

		ProjectName:   projectName,
		ShootName:     shootName,
		Landscape:     landscape,
		Cloudprovider: cloudprovider,
		K8sVersion:    k8sVersion,
		BOM:           bom,
	}

	testrunner.Run(config, parameters)
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

	if os.Getenv("LOG_LEVEL") == "debug" {
		log.SetLevel(log.DebugLevel)
		log.Warn("Set debug log level")
	}

	// Set commandline flags
	flag.StringVar(&gardenKubeconfigPath, "gardener-kubeconfig-path", "", "Path to the gardener kubeconfig.")
	flag.StringVar(&tmKubeconfigPath, "tm-kubeconfig-path", "", "Path to the testmachinery cluster kubeconfig")
	flag.Int64Var(&timeout, "timeout", -1, "timout of the testrunner to wait for the complete testrun to finish.")
	flag.StringVar(&testrunChartPath, "testruns-chart-path", "", "Path to the testruns chart.")

	flag.StringVar(&projectName, "project-name", "", "Gardener project name of the shoot")
	flag.StringVar(&shootName, "shoot-name", "", "Shoot name which is used to run tests.")

	flag.StringVar(&landscape, "landscape", "", "Current gardener landscape.")
	flag.StringVar(&cloudprovider, "cloudprovider", "", "Cloudprovider where the shoot is created.")
	flag.StringVar(&cloudprofile, "cloudprofile", "", "Cloudprofile of shoot.")
	flag.StringVar(&secretBinding, "secret-binding", "", "SecretBinding that should be used to create the shoot.")
	flag.StringVar(&region, "region", "", "Region where the shoot is created.")
	flag.StringVar(&zone, "zone", "", "Zone of the shoot worker nodes. Not required for azure shoots.")
	flag.StringVar(&k8sVersion, "k8s-version", "", "Kubernetes version of the shoot.")
	flag.StringVar(&bom, "bom", "", "Component versions of the currently deployed landscape.")

	flag.StringVar(&outputFilePath, "output-file-path", "", "The filepath where the summary should be written to.")
	flag.StringVar(&elasticSearchConfigName, "es-config-name", "", "The elasticsearch secret-server config name.")
	flag.StringVar(&s3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")

}
