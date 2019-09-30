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
	"github.com/gardener/test-infra/pkg/testrunner"
	errors "github.com/gardener/test-infra/pkg/testrunner/error"
	"github.com/gardener/test-infra/pkg/testrunner/renderer"
	"github.com/go-logr/logr"
	"os"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
)

// RenderShootTestruns renders a helm chart with containing testruns, adds the provided parameters and values, and returns the parsed and modified testruns.
// Adds the component descriptor to metadata.
func RenderShootTestruns(log logr.Logger, tmClient kubernetes.Interface, parameters *ShootTestrunParameters, metadata *testrunner.Metadata) (ShootRunList, error) {

	versions, err := getK8sVersions(parameters)
	if err != nil {
		log.Error(err, "cannot get kubernetes versions")
		os.Exit(1)
	}

	componentDescriptor, err := componentdescriptor.GetComponentsFromFile(parameters.ComponentDescriptorPath)
	if err != nil {
		return nil, fmt.Errorf("cannot decode and parse the component descriptor: %s", err.Error())
	}
	metadata.ComponentDescriptor = componentDescriptor.JSON()
	exposeGardenerVersionToParameters(componentDescriptor, parameters)

	renderedTestruns, err := RenderChart(log, tmClient, parameters, versions)
	if err != nil {
		return nil, err
	}

	// parse the rendered testruns and add locations from BOM of a bom was provided.
	testruns := make(ShootRunList, 0)
	for _, rendered := range renderedTestruns {
		testrunMetadata := *metadata
		testrunMetadata.KubernetesVersion = rendered.Metadata.KubernetesVersion

		// Add all repositories defined in the component descriptor to the testrun locations.
		// This gives us all dependent repositories as well as there deployed version.
		if err := renderer.AddBOMLocationsToTestrun(rendered.testrun, "default", componentDescriptor, true); err != nil {
			log.Info(fmt.Sprintf("cannot add bom locations: %s", err.Error()))
			continue
		}

		// Add runtime annotations to the testrun
		addAnnotationsToTestrun(rendered.testrun, testrunMetadata.CreateAnnotations())

		testruns = append(testruns, &ShootRun{
			Parameters: rendered.Parameters,
			Run: &testrunner.Run{
				Testrun:  rendered.testrun,
				Metadata: &testrunMetadata,
			},
		})
	}

	if len(testruns) == 0 {
		return nil, errors.NewNotRenderedError(fmt.Sprintf("no testruns in the helm chart at %s", parameters.TestrunChartPath))
	}

	return testruns, nil
}

func exposeGardenerVersionToParameters(componentDescriptor componentdescriptor.ComponentList, parameters *ShootTestrunParameters) {
	parameters.GardenerVersion = getGardenerVersionFromComponentDescriptor(componentDescriptor)
}
