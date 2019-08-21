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

// ShootTestrunParameters are the parameters which describe the test that is executed by the testrunner.
type ShootTestrunParameters struct {
	// Path to the kubeconfig where the gardener is running.
	GardenKubeconfigPath string
	Namespace            string
	TestrunChartPath     string
	// Ignore K8sVersion and generate a testrun for every valid version that is defined in the cloudprofile
	MakeVersionMatrix bool

	ProjectName             string
	ShootName               string
	Landscape               string
	Cloudprovider           string
	Cloudprofile            string
	SecretBinding           string
	Region                  string
	Zone                    string
	K8sVersion              string
	MachineType             string
	MachineImage            string
	MachineImageVersion     string
	AutoscalerMin           string
	AutoscalerMax           string
	FloatingPoolName        string
	LoadBalancerProvider    string
	ComponentDescriptorPath string

	GardenerVersion string
	SetValues       string
	FileValues      []string
}

type GardenerTestrunParameters struct {
	Namespace        string
	TestrunChartPath string

	ComponentDescriptorPath         string
	UpgradedComponentDescriptorPath string

	Landscape                string
	GardenerCurrentVersion   string
	GardenerCurrentRevision  string
	GardenerUpgradedVersion  string
	GardenerUpgradedRevision string

	SetValues  string
	FileValues []string
}

// TestrunFile is the internal representation of a rendered testrun chart with metadata information
type TestrunFile struct {
	File     string
	Metadata TestrunFileMetadata
}

// TestrunFileMetadata represents the metadata of a rendered testrun.
type TestrunFileMetadata struct {
	KubernetesVersion string
}
