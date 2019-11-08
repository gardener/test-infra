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
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	trerrors "github.com/gardener/test-infra/pkg/testrunner/error"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"io/ioutil"
)

// RenderShootTestruns renders a helm chart with containing testruns, adds the provided parameters and values, and returns the parsed and modified testruns.
// Adds the component descriptor to metadata.
func RenderTestruns(log logr.Logger, parameters *Parameters, shootFlavors []*common.ExtendedShoot) (testrunner.RunList, error) {
	log.V(3).Info(fmt.Sprintf("Parameters: %+v", util.PrettyPrintStruct(parameters)))

	getInternalParameters, err := getInternalParametersFunc(parameters)
	if err != nil {
		return nil, err
	}

	tmplRenderer, err := newRenderer(log, parameters.SetValues, parameters.FileValues)
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
			ComponentDescriptor: componentDescriptor,
			ChartPath:           chartPath,
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
	values := map[string]interface{}{
		"gardener": map[string]interface{}{
			"version": parameters.GardenerVersion,
		},
		"kubeconfigs": map[string]interface{}{
			"gardener": string(parameters.GardenerKubeconfig),
		},
	}

	metadata := &testrunner.Metadata{
		Landscape:           parameters.Landscape,
		ComponentDescriptor: parameters.ComponentDescriptor.JSON(),
	}

	return renderer.RenderChart(parameters, parameters.ChartPath, values, metadata, nil)
}

func renderChartWithShoot(log logr.Logger, renderer *templateRenderer, parameters *internalParameters, shootFlavors []*common.ExtendedShoot) (testrunner.RunList, error) {
	runs := make(testrunner.RunList, 0)
	if parameters.ChartPath == "" {
		return runs, nil
	}

	for _, shoot := range shootFlavors {
		chartPath, err := determineAbsoluteShootChartPath(parameters.ChartPath, shoot.ChartPath)
		if err != nil {
			return nil, errors.Wrap(err, "unable to determine chart to render")
		}
		workers, err := encodeRawObject(shoot.Workers)
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse worker config")
		}
		log.V(3).Info(fmt.Sprintf("Workers: \n%s \n", util.PrettyPrintStruct(workers)))

		infrastructure, err := encodeRawObject(shoot.InfrastructureConfig)
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse infrastructure config")
		}
		log.V(3).Info(fmt.Sprintf("Infrastructure: \n%s \n", util.PrettyPrintStruct(infrastructure)))

		controlplane, err := encodeRawObject(shoot.ControlPlaneConfig)
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse infrastructure config")
		}
		log.V(3).Info(fmt.Sprintf("Controlplane: \n%s \n", util.PrettyPrintStruct(controlplane)))

		prevPrePatchVersion, prevPatchVersion, err := util.GetPreviousKubernetesVersions(shoot.Cloudprofile, shoot.KubernetesVersion)
		if err != nil {
			log.Info("unable to get previous versions", "error", err.Error())
		}

		values := map[string]interface{}{
			"shoot": map[string]interface{}{
				"name":                   shoot.Name,
				"projectNamespace":       shoot.Namespace,
				"cloudprovider":          shoot.Provider,
				"cloudprofile":           shoot.CloudprofileName,
				"secretBinding":          shoot.SecretBinding,
				"region":                 shoot.Region,
				"zone":                   shoot.Zone,
				"workers":                workers,
				"k8sVersion":             shoot.KubernetesVersion.Version,
				"k8sPrevPrePatchVersion": prevPrePatchVersion.Version,
				"k8sPrevPatchVersion":    prevPatchVersion.Version,
				"floatingPoolName":       shoot.FloatingPoolName,
				"loadbalancerProvider":   shoot.LoadbalancerProvider,
				"infrastructureConfig":   infrastructure,
				"controlplaneConfig":     controlplane,
			},
			"gardener": map[string]interface{}{
				"version": parameters.GardenerVersion,
			},
			"kubeconfigs": map[string]interface{}{
				"gardener": string(parameters.GardenerKubeconfig),
			},
		}

		metadata := &testrunner.Metadata{
			FlavorDescription:   shoot.Description,
			Landscape:           parameters.Landscape,
			ComponentDescriptor: parameters.ComponentDescriptor.JSON(),
			CloudProvider:       string(shoot.Provider),
			KubernetesVersion:   shoot.KubernetesVersion.Version,
			Region:              shoot.Region,
			Zone:                shoot.Zone,
			OperatingSystem:     shoot.Workers[0].Machine.Image.Name, // todo: check if there a possible multiple workerpools with different images
		}

		shootRuns, err := renderer.RenderChart(parameters, chartPath, values, metadata, shoot)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to render chart for shoot %v", shoot)
		}
		runs = append(runs, shootRuns...)
	}
	return runs, nil
}
