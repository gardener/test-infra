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
	"github.com/gardener/test-infra/pkg/shootflavors"
	"io/ioutil"

	trerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

// RenderShootTestruns renders a helm chart with containing testruns, adds the provided parameters and values, and returns the parsed and modified testruns.
// Adds the component descriptor to metadata.
func RenderTestruns(log logr.Logger, parameters *Parameters, shootFlavors []*shootflavors.ExtendedFlavorInstance) (testrunner.RunList, error) {
	log.V(3).Info(fmt.Sprintf("Parameters: %+v", util.PrettyPrintStruct(parameters)))

	getInternalParameters, err := getInternalParametersFunc(parameters)
	if err != nil {
		return nil, err
	}

	tmplRenderer, err := newTemplateRenderer(log, parameters.SetValues, parameters.FileValues)
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize template renderer")
	}

	runs, err := renderDefaultChart(tmplRenderer, getInternalParameters(parameters.DefaultTestrunChartPath))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to render default chart from %s", parameters.DefaultTestrunChartPath)
	}

	shootRuns, err := renderChartWithShoot(log, tmplRenderer, getInternalParameters(parameters.FlavoredTestrunChartPath), shootFlavors)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to render shoot chart from %s", parameters.FlavoredTestrunChartPath)
	}
	runs = append(runs, shootRuns...)

	if len(runs) == 0 {
		return nil, trerrors.NewNotRenderedError(fmt.Sprintf("no testruns in the helm chart at %s or %s", parameters.DefaultTestrunChartPath, parameters.FlavoredTestrunChartPath))
	}

	return runs, nil
}

func getInternalParametersFunc(parameters *Parameters) (func(string) *internalParameters, error) {
	componentDescriptor, err := componentdescriptor.GetComponentsFromFile(parameters.ComponentDescriptorPath)
	if err != nil {
		return nil, fmt.Errorf("cannot decode and parse the component descriptor: %s", err.Error())
	}
	gardenerKubeconfig, err := ioutil.ReadFile(parameters.GardenKubeconfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read gardener kubeconfig %s", parameters.GardenKubeconfigPath)
	}
	gardenerVersion := getGardenerVersionFromComponentDescriptor(componentDescriptor)

	return func(chartPath string) *internalParameters {
		return &internalParameters{
			FlavorConfigPath:    parameters.FlavorConfigPath,
			ComponentDescriptor: componentDescriptor,
			ChartPath:           chartPath,
			Namespace:           parameters.Namespace,
			GardenerKubeconfig:  gardenerKubeconfig,
			GardenerVersion:     gardenerVersion,
			Landscape:           parameters.Landscape,
		}
	}, nil
}

func renderDefaultChart(renderer *templateRenderer, parameters *internalParameters) (testrunner.RunList, error) {
	if parameters.ChartPath == "" {
		return make(testrunner.RunList, 0), nil
	}
	return renderer.Render(parameters, parameters.ChartPath, NewDefaultValueRenderer(parameters))
}

func renderChartWithShoot(log logr.Logger, renderer *templateRenderer, parameters *internalParameters, shootFlavors []*shootflavors.ExtendedFlavorInstance) (testrunner.RunList, error) {
	runs := make(testrunner.RunList, 0)
	if parameters.ChartPath == "" {
		return runs, nil
	}

	for _, flavor := range shootFlavors {
		chartPath, err := determineAbsoluteShootChartPath(parameters, flavor.Get().ChartPath)
		if err != nil {
			return nil, errors.Wrap(err, "unable to determine chart to render")
		}

		valueRenderer := NewShootValueRenderer(log, flavor, parameters)
		shootRuns, err := renderer.Render(parameters, chartPath, valueRenderer)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to render chart for flavor %v", flavor)
		}
		runs = append(runs, shootRuns...)
	}
	return runs, nil
}
