// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
)

// Parameters are the parameters which describe the test that is executed by the testrunner.
type Parameters struct {
	FlavorConfigPath string
	// Path to the kubeconfig where the gardener is running.
	GardenKubeconfigPath     string
	Namespace                string
	FlavoredTestrunChartPath string
	DefaultTestrunChartPath  string

	// metadata
	Landscape               string
	ComponentDescriptorPath string
	Repository              string
	OCMConfigPath           string

	SetValues  []string
	FileValues []string
}

type internalParameters struct {
	FlavorConfigPath    string
	ComponentDescriptor componentdescriptor.ComponentList
	ChartPath           string
	Namespace           string

	GardenerKubeconfig []byte
	GardenerVersion    string
	Landscape          string

	AdditionalLocations []common.AdditionalLocation
}

func (i *internalParameters) DeepCopy() *internalParameters {
	return &internalParameters{
		FlavorConfigPath:    i.FlavorConfigPath,
		ComponentDescriptor: i.ComponentDescriptor,
		ChartPath:           i.ChartPath,
		Namespace:           i.Namespace,
		GardenerKubeconfig:  i.GardenerKubeconfig,
		GardenerVersion:     i.GardenerVersion,
		Landscape:           i.Landscape,
	}
}

// ValueRenderer renders the helm values, run metadata and info for a specific chart rendering
type ValueRenderer interface {
	Render(defaultValues map[string]interface{}) (values map[string]interface{}, metadata *metadata.Metadata, info interface{}, err error)
}
