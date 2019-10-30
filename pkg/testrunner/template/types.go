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
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner"
)

// ShootTestrunParameters are the parameters which describe the test that is executed by the testrunner.
type ShootTestrunParameters struct {
	// Path to the kubeconfig where the gardener is running.
	GardenKubeconfigPath  string
	Namespace             string
	ShootTestrunChartPath string
	TestrunChartPath      string

	ShootName string

	// metadata
	Landscape               string
	ComponentDescriptorPath string
	GardenerVersion         string

	SetValues  string
	FileValues []string
}

// RenderedTestrun is the internal representation of a rendered testrun chart with metadata information
type RenderedTestrun struct {
	testrun    *v1beta1.Testrun
	Parameters ShootTestrunParameters
	Metadata   testrunner.Metadata
}

// TestrunFileMetadata represents the metadata of a rendered testrun.
type TestrunFileMetadata struct {
	KubernetesVersion string
}

// ShootRun represents a testrun where one shoot is tested
// with its rendered testrun and configuration
type ShootRun struct {
	Run        *testrunner.Run
	Parameters ShootTestrunParameters
}
type ShootRunList []*ShootRun
