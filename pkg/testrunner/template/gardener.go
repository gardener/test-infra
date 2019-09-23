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
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	errors "github.com/gardener/test-infra/pkg/testrunner/error"
	"github.com/gardener/test-infra/pkg/testrunner/renderer"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
)

// RenderGardenerTestrun renders a helm chart with containing testruns.
// The current component_descriptor as well as the upgraded component_descriptor are added to the locationSets.
func RenderGardenerTestrun(log logr.Logger, tmClient kubernetes.Interface, parameters *GardenerTestrunParameters, metadata *testrunner.Metadata) (testrunner.RunList, error) {
	var componentDescriptor componentdescriptor.ComponentList

	componentDescriptor, err := componentdescriptor.GetComponentsFromFile(parameters.ComponentDescriptorPath)
	if err != nil {
		return nil, fmt.Errorf("cannot decode and parse the component descriptor: %s", err.Error())
	}
	metadata.ComponentDescriptor = componentDescriptor.JSON()
	upgradedComponentDescriptor, err := componentdescriptor.GetComponentsFromFile(parameters.UpgradedComponentDescriptorPath)
	if err != nil {
		return nil, fmt.Errorf("cannot decode and parse the component descriptor: %s", err.Error())
	}
	metadata.UpgradedComponentDescriptor = upgradedComponentDescriptor.JSON()

	tmChartRenderer, err := chartrenderer.NewForConfig(tmClient.RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("cannot create chartrenderer for tm cluster: %s", err.Error())
	}

	values := map[string]interface{}{
		"gardener": map[string]interface{}{
			"current": map[string]string{
				"version":  util.StringDefault(parameters.GardenerCurrentVersion, getGardenerVersionFromComponentDescriptor(componentDescriptor)),
				"revision": util.StringDefault(parameters.GardenerCurrentRevision, getGardenerVersionFromComponentDescriptor(componentDescriptor)),
			},
			"upgraded": map[string]string{
				"version":  util.StringDefault(parameters.GardenerUpgradedVersion, getGardenerVersionFromComponentDescriptor(upgradedComponentDescriptor)),
				"revision": util.StringDefault(parameters.GardenerUpgradedRevision, getGardenerVersionFromComponentDescriptor(upgradedComponentDescriptor)),
			},
		},
	}
	values, err = determineValues(values, parameters.SetValues, parameters.FileValues)
	if err != nil {
		return nil, err
	}
	log.V(3).Info(fmt.Sprintf("Values: \n%s \n", util.PrettyPrintStruct(values)))

	chart, err := tmChartRenderer.Render(parameters.TestrunChartPath, "", parameters.Namespace, values)
	if err != nil {
		return nil, fmt.Errorf("cannot render chart: %s", err.Error())
	}

	files := ParseTestrunChart(chart, TestrunFileMetadata{})

	// parse the rendered testruns and add locations from BOM of a bom was provided.
	testruns := make([]*testrunner.Run, 0)
	for _, file := range files {
		tr, err := util.ParseTestrun([]byte(file.File))
		if err != nil {
			log.Info(fmt.Sprintf("cannot parse rendered file: %s", err.Error()))
			continue
		}

		testrunMetadata := *metadata

		// Add all repositories defined in the component descriptor to the testrun locations.
		// This gives us all dependent repositories as well as there deployed version.
		if err := renderer.AddBOMLocationsToTestrun(&tr, "default", componentDescriptor, true); err != nil {
			log.Info(fmt.Sprintf("cannot add bom locations: %s", err.Error()))
			continue
		}
		if err := renderer.AddBOMLocationsToTestrun(&tr, "upgraded", upgradedComponentDescriptor, false); err != nil {
			log.Info(fmt.Sprintf("cannot add bom locations: %s", err.Error()))
			continue
		}

		// Add runtime annotations to the testrun
		addAnnotationsToTestrun(&tr, metadata.CreateAnnotations())

		testruns = append(testruns, &testrunner.Run{
			Testrun:  &tr,
			Metadata: &testrunMetadata,
		})
	}

	if len(testruns) == 0 {
		return nil, errors.NewNotRenderedError(fmt.Sprintf("no testruns in the helm chart at %s", parameters.TestrunChartPath))
	}

	return testruns, nil
}
