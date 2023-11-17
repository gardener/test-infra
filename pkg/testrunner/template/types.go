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
