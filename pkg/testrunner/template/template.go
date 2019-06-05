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

	"github.com/gardener/test-infra/pkg/testrunner"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util"
	log "github.com/sirupsen/logrus"
)

// Render renders a helm chart with containing testruns, adds the provided parameters and values, and returns the parsed and modified testruns.
// Adds the component descriptor to metadata.
func Render(tmClient kubernetes.Interface, parameters *TestrunParameters, metadata *testrunner.Metadata) (testrunner.RunList, error) {
	var componentDescriptor componentdescriptor.ComponentList

	versions, err := getK8sVersions(parameters)
	if err != nil {
		log.Fatal(err.Error())
	}

	if parameters.ComponentDescriptorPath != "" {
		data, err := ioutil.ReadFile(parameters.ComponentDescriptorPath)
		if err != nil {
			return nil, fmt.Errorf("Cannot read component descriptor file %s: %s", parameters.ComponentDescriptorPath, err.Error())
		}
		componentDescriptor, err = componentdescriptor.GetComponents(data)
		if err != nil {
			return nil, fmt.Errorf("Cannot decode and parse the component descriptor: %s", err.Error())
		}
		metadata.ComponentDescriptor = componentDescriptor.JSON()
		exposeGardenerVersionToParameters(componentDescriptor, parameters)
	}

	files, err := RenderChart(tmClient, parameters, versions)
	if err != nil {
		return nil, err
	}

	// parse the rendered testruns and add locations from BOM of a bom was provided.
	testruns := []*testrunner.Run{}
	for _, file := range files {
		tr, err := util.ParseTestrun([]byte(file.File))
		if err != nil {
			log.Warnf("Cannot parse rendered file: %s", err.Error())
		}

		testrunMetadata := *metadata
		testrunMetadata.KubernetesVersion = file.Metadata.KubernetesVersion

		// Add all repositories defined in the component descriptor to the testrun locations.
		// This gives us all dependent repositories as well as there deployed version.
		addBOMLocationsToTestrun(&tr, componentDescriptor)

		// Add runtime annotations to the testrun
		addAnnotationsToTestrun(&tr, metadata.Annotations())

		testruns = append(testruns, &testrunner.Run{
			Testrun:  &tr,
			Metadata: &testrunMetadata,
		})
	}

	if len(testruns) == 0 {
		return nil, fmt.Errorf("No testruns in the helm chart at %s", parameters.TestrunChartPath)
	}

	return testruns, nil
}

func exposeGardenerVersionToParameters(componentDescriptor componentdescriptor.ComponentList, parameters *TestrunParameters) {
	for _, component := range componentDescriptor {
		if component.Name == "github.com/gardener/gardener.version" {
			parameters.GardenerVersion = component.Version
			return
		}
	}
}
