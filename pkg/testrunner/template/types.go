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
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
)

// Parameters are the parameters which describe the test that is executed by the testrunner.
type Parameters struct {
	// Path to the kubeconfig where the gardener is running.
	GardenKubeconfigPath  string
	Namespace             string
	ShootTestrunChartPath string
	TestrunChartPath      string

	// metadata
	Landscape               string
	ComponentDescriptorPath string

	SetValues  string
	FileValues []string
}

type internalParameters struct {
	ComponentDescriptor componentdescriptor.ComponentList
	ChartPath           string

	GardenerKubeconfig []byte
	GardenerVersion    string
	Landscape          string
}
