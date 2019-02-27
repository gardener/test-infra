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
	"io/ioutil"
	"os"
	"sync"

	"github.com/gardener/gardener/pkg/chartrenderer"

	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"k8s.io/client-go/tools/clientcmd"

	tmclientset "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned"
	log "github.com/sirupsen/logrus"
)

var (
	namespace                 = "default"
	maxWaitTimeSeconds  int64 = 3600
	pollIntervalSeconds int64 = 60
)

// Run renders a testrun, deploys it to a testmachinery cluster and waits for the testruns results
func Run(config *TestrunConfig, parameters *TestrunParameters) {
	log.Info("Get Testmachinery clients")

	log.Infof("Config: %+v", util.PrettyPrintStruct(config))
	log.Infof("Parameters: %+v", util.PrettyPrintStruct(parameters))

	if config.Timeout != nil && *config.Timeout > -1 {
		maxWaitTimeSeconds = *config.Timeout
	}

	metadata := &Metadata{
		Landscape:         parameters.Landscape,
		CloudProvider:     parameters.Cloudprovider,
		KubernetesVersion: parameters.K8sVersion,
	}
	if parameters.ComponentDescriptorPath != "" {
		data, err := ioutil.ReadFile(parameters.ComponentDescriptorPath)
		if err != nil {
			log.Warnf("Cannot read component descriptor file %s: %s", parameters.ComponentDescriptorPath, err.Error())
		}
		components, err := componentdescriptor.GetComponents(data)
		if err != nil {
			log.Warnf("Cannot decode and parse BOM %s", err.Error())
		} else {
			metadata.BOM = components
		}
	}

	tmConfig, err := clientcmd.BuildConfigFromFlags("", config.TmKubeconfigPath)
	tmClient := tmclientset.NewForConfigOrDie(tmConfig)

	chart, err := renderChart(config, parameters)
	if err != nil {
		log.Fatalf("Cannot render chart: %s", err.Error())
	}

	finishedTestruns := runChart(tmClient, chart, parameters, metadata.BOM)

	testrunsFailed := false
	for _, tr := range finishedTestruns {
		err = Output(config, tr, metadata)
		if err != nil {
			log.Fatal(err.Error())
		}

		err = PersistFile(config.OutputFile, config.ESConfigName)
		if err != nil {
			log.Errorf("Cannot persist file %s: %s", config.OutputFile, err.Error())
			return
		}

		if tr.Status.Phase == argov1.NodeSucceeded {
			log.Infof("The testrun %s finished successfully", tr.Name)
		} else {
			testrunsFailed = true
			log.Errorf("Testrun %s failed with phase %s", tr.Name, tr.Status.Phase)
		}
	}

	GenerateNotificationConfigForAlerting(finishedTestruns, config.ConcourseOnErrorDir)

	log.Info("Testrunner finished.")
	if testrunsFailed {
		os.Exit(1)
	}
}

// runChart tries to parse each rendered file of a chart into a testrun.
// If a filecontent is a testrun then it is deployed into the testmachinery.
func runChart(tmClient *tmclientset.Clientset, chart *chartrenderer.RenderedChart, parameters *TestrunParameters, bom []*componentdescriptor.Component) []*tmv1beta1.Testrun {
	var wg sync.WaitGroup
	mutex := &sync.Mutex{}
	finishedTestruns := []*tmv1beta1.Testrun{}
	for fileName, fileContent := range chart.Files {
		tr, err := util.ParseTestrun([]byte(fileContent))
		if err != nil {
			log.Warnf("Cannot parse %s: %s", fileName, err.Error())
		}

		// Add current dependency repositories to the testrun location.
		// This gives us all dependent repositories as well as there deployed version.
		addBOMLocationsToTestrun(&tr, bom)

		wg.Add(1)
		go func(tr *tmv1beta1.Testrun) {
			defer wg.Done()
			tr, err := runTestrun(tmClient, tr, parameters)
			if err != nil {
				log.Error(err.Error())
				tr.Status.Phase = argov1.NodeFailed
			}
			mutex.Lock()
			finishedTestruns = append(finishedTestruns, tr)
			mutex.Unlock()
		}(&tr)
	}
	wg.Wait()
	log.Infof("All testruns completed.")
	return finishedTestruns
}
