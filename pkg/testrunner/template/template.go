// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	trerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/shootflavors"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util"
)

// RenderTestruns renders a helm chart with containing testruns, adds the provided parameters and values, and returns the parsed and modified testruns.
// Adds the component descriptor to metadata.
func RenderTestruns(ctx context.Context, log logr.Logger, parameters *Parameters, shootFlavors []*shootflavors.ExtendedFlavorInstance) (testrunner.RunList, error) {
	log.V(3).Info(fmt.Sprintf("Parameters: %+v", util.PrettyPrintStruct(parameters)))

	getInternalParameters, err := getInternalParametersFunc(ctx, log, parameters)
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

func getInternalParametersFunc(ctx context.Context, log logr.Logger, parameters *Parameters) (func(string) *internalParameters, error) {
	components, err := componentdescriptor.GetComponents(ctx, log, parameters.ComponentDescriptorPath, parameters.Repository, func(opts *componentdescriptor.Options) {
		opts.CfgPath = parameters.OCMConfigPath
	})
	if err != nil {
		return nil, err
	}
	var gardenerKubeconfig []byte
	if len(parameters.GardenKubeconfigPath) != 0 {
		gardenerKubeconfig, err = os.ReadFile(parameters.GardenKubeconfigPath)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot read gardener kubeconfig %s", parameters.GardenKubeconfigPath)
		}
	}
	gardenerVersion := getGardenerVersionFromComponentDescriptor(components)

	return func(chartPath string) *internalParameters {
		return &internalParameters{
			FlavorConfigPath:    parameters.FlavorConfigPath,
			ComponentDescriptor: components,
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
		parameters := parameters.DeepCopy()
		parameters.AdditionalLocations = flavor.Get().AdditionalLocations
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
