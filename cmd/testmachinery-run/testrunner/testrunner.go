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

package testrunner

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gardener/gardener/pkg/operation/common"
	tmclientset "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	namespace = "default"
	// outputFilePath is the path where the testresult is written to.
	outputFilePath            = "./testout"
	maxWaitTimeSeconds  int64 = 3600
	pollIntervalSeconds int64 = 60
	esCfgName                 = "sap_internal"
)

// Run renders a testrun, deploys it to a testmachinery cluster and waits for the testruns results
func Run(config *TestrunConfig, parameters *TestrunParameters) {
	log.Info("Get Testmachinery clients")

	if config.ESConfigName == "" {
		config.ESConfigName = esCfgName
	}
	if config.OutputFile == "" {
		config.OutputFile = outputFilePath
	}
	if config.Timeout != nil && *config.Timeout > -1 {
		maxWaitTimeSeconds = *config.Timeout
	}

	metadata := &Metadata{
		Landscape:         parameters.Landscape,
		CloudProvider:     parameters.Cloudprovider,
		KubernetesVersion: parameters.K8sVersion,
	}
	if parameters.BOM != "" {
		var jsonBody interface{}
		err := json.Unmarshal([]byte(parameters.BOM), &jsonBody)
		if err != nil {
			log.Warnf("Cannot decode BOM %s", err.Error())
		} else {
			metadata.BOM = jsonBody
		}
	}

	tmConfig, err := clientcmd.BuildConfigFromFlags("", config.TmKubeconfigPath)
	tmClient := tmclientset.NewForConfigOrDie(tmConfig)

	tmClusterClient, err := kubernetes.NewClientFromFile(config.TmKubeconfigPath, nil, client.Options{})
	if err != nil {
		log.Fatalf("couldn't create k8s client from kubeconfig filepath %s: %v", config.TmKubeconfigPath, err)
	}
	tmChartRenderer, err := chartrenderer.New(tmClusterClient)
	if err != nil {
		log.Fatalf("Cannot create chartrenderer for gardener  %s", err.Error())
	}

	gardenKubeconfig, err := ioutil.ReadFile(config.GardenKubeconfigPath)
	if err != nil {
		log.Fatalf("Cannot read gardener kubeconfig %s, Error: %s", config.GardenKubeconfigPath, err.Error())
	}

	log.Infof("Deploying testrun %s", parameters.TestrunName)

	tmChartRenderer, err = chartrenderer.New(tmClusterClient)
	if err != nil {
		log.Fatalf("Cannot create chartrenderer for gardener  %s", err.Error())
	}

	err = common.ApplyChart(tmClusterClient, tmChartRenderer, parameters.TestrunChartPath, parameters.TestrunName, namespace, map[string]interface{}{
		"testrunName": parameters.TestrunName,
		"shoot": map[string]interface{}{
			"name":             parameters.ShootName,
			"projectNamespace": fmt.Sprintf("garden-%s", parameters.ProjectName),
			"cloudprovider":    parameters.Cloudprovider,
			"k8sVersion":       parameters.K8sVersion,
		},
		"kubeconfigs": map[string]interface{}{
			"gardener": string(gardenKubeconfig),
		},
	}, map[string]interface{}{})
	if err != nil {
		log.Fatalf("Cannot render chart %s", err.Error())
	}

	log.Infof("Testrun %s deployed", parameters.TestrunName)

	var tr *tmv1beta1.Testrun
	var testrunPhase argov1.NodePhase
	startTime := time.Now()
	for util.Completed(testrunPhase) {
		var err error

		if util.MaxTimeExceeded(startTime, maxWaitTimeSeconds) {
			log.Fatalf("Maximum wait time of %d is exceeded by Testrun %s", maxWaitTimeSeconds, parameters.TestrunName)
		}

		tr, err = tmClient.Testmachinery().Testruns("default").Get(parameters.TestrunName, metav1.GetOptions{})
		if err != nil {
			log.Errorf("Cannot get testrun: %s", err.Error())
		}
		testrunPhase = tr.Status.Phase

		log.Infof("Testrun %s is %s. Waiting ...", parameters.TestrunName, testrunPhase)

		time.Sleep(time.Duration(pollIntervalSeconds) * time.Second)
	}

	err = Output(config, tr, metadata)
	if err != nil {
		log.Fatal(err.Error())
	}

	if testrunPhase == argov1.NodeSucceeded {
		log.Info("The testrun finished successfully")
	} else {
		log.Errorf("Testrun failed with phase %s", testrunPhase)
	}

	err = PersistFile(outputFilePath)
	if err != nil {
		log.Errorf("Cannot persist file %s: %s", config.OutputFile, err.Error())
		return
	}

	log.Info("Testrunner finished.")
	if testrunPhase != argov1.NodeSucceeded {
		os.Exit(1)
	}
}
