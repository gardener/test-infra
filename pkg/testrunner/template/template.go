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

package template

import (
	"fmt"
	"io/ioutil"

	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	"github.com/gardener/test-infra/pkg/util"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Render renders a helm chart with containing testruns, adds the provided parameters and values, and returns the parsed and modified testruns.
func Render(tmKubeconfigPath string, parameters *TestrunParameters, metadata *result.Metadata) ([]*tmv1beta1.Testrun, error) {

	// if the kubernetes version is not set, get the latest version defined by the cloudprofile
	if parameters.K8sVersion == "" {
		var err error
		parameters.K8sVersion, err = getLatestK8sVersion(parameters.GardenKubeconfigPath, parameters.Cloudprofile, parameters.Cloudprovider)
		if err != nil {
			log.Fatalf("Kubernetes is not defined nor can it be read from the cloudprofile: %s", err.Error())
		}
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

	chart, err := RenderChart(tmKubeconfigPath, parameters)
	if err != nil {
		return nil, err
	}

	// parse the rendered testruns and add locations from BOM of a bom was provided.
	testruns := []*tmv1beta1.Testrun{}
	for fileName, fileContent := range chart.Files {
		tr, err := util.ParseTestrun([]byte(fileContent))
		if err != nil {
			log.Warnf("Cannot parse %s: %s", fileName, err.Error())
		}

		// Add current dependency repositories to the testrun location.
		// This gives us all dependent repositories as well as there deployed version.
		addBOMLocationsToTestrun(&tr, metadata.BOM)
		testruns = append(testruns, &tr)
	}

	if len(testruns) == 0 {
		return nil, fmt.Errorf("No testruns in the helm chart at %s", parameters.TestrunChartPath)
	}

	return testruns, nil
}

// RenderChart renders the provided helm chart with testruns and adds the testrun parameters.
func RenderChart(tmKubeconfigPath string, parameters *TestrunParameters) (*chartrenderer.RenderedChart, error) {
	log.Debugf("Parameters: %+v", util.PrettyPrintStruct(parameters))
	log.Debugf("Render chart from %s", parameters.TestrunChartPath)

	tmClusterClient, err := kubernetes.NewClientFromFile(tmKubeconfigPath, nil, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("couldn't create k8s client from kubeconfig filepath %s: %v", tmKubeconfigPath, err)
	}
	tmChartRenderer, err := chartrenderer.New(tmClusterClient)
	if err != nil {
		return nil, fmt.Errorf("Cannot create chartrenderer for gardener  %s", err.Error())
	}

	gardenKubeconfig, err := ioutil.ReadFile(parameters.GardenKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot read gardener kubeconfig %s, Error: %s", parameters.GardenKubeconfigPath, err.Error())
	}

	return tmChartRenderer.Render(parameters.TestrunChartPath, "", parameters.Namespace, map[string]interface{}{
		"shoot": map[string]interface{}{
			"name":             fmt.Sprintf("%s-%s", parameters.ShootName, util.RandomString(5)),
			"projectNamespace": fmt.Sprintf("garden-%s", parameters.ProjectName),
			"cloudprovider":    parameters.Cloudprovider,
			"cloudprofile":     parameters.Cloudprofile,
			"secretBinding":    parameters.SecretBinding,
			"region":           parameters.Region,
			"zone":             parameters.Zone,
			"k8sVersion":       parameters.K8sVersion,
			"machinetype":      parameters.MachineType,
			"autoscalerMin":    parameters.AutoscalerMin,
			"autoscalerMax":    parameters.AutoscalerMax,
			"floatingPoolName": parameters.FloatingPoolName,
		},
		"kubeconfigs": map[string]interface{}{
			"gardener": string(gardenKubeconfig),
		},
	})
}
