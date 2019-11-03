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

package common

import (
	gardenv1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
)

// ExtendedShootFlavors contains a list of extended shoot flavors
type ExtendedShootFlavors struct {
	Flavors []*ExtendedShootFlavor `json:"flavors"`
}

// ExtendedShoot is one instance that is generated from a extended shoot flavor
type ExtendedShoot struct {
	Shoot
	ExtendedShootConfiguration
}

// ExtendedShootFlavor is the shoot flavor with extended configuration
type ExtendedShootFlavor struct {
	ShootFlavor
	ExtendedConfiguration
}

// Shoot is one instance that is generated from a shoot flavor
type Shoot struct {
	// Cloudprovider of the shoot
	Provider CloudProvider

	// Kubernetes versions to test
	KubernetesVersion gardenv1alpha1.ExpirableVersion

	// Worker pools to test
	Workers []gardenv1alpha1.Worker
}

// ShootFlavor describes the shoot flavors that should be tested.
type ShootFlavor struct {
	// Cloudprovider of the shoot
	Provider CloudProvider `json:"provider"`

	// Kubernetes versions to test
	KubernetesVersions ShootKubernetesVersionFlavor `json:"kubernetes"`

	// Worker pools to test
	Workers []ShootWorkerFlavor `json:"workers"`
}

type ShootKubernetesVersionFlavor struct {
	// Regex to select versions from the cloudprofile
	// +optional
	Pattern *string `json:"pattern"`

	// List of versions to test
	// +optional
	Versions *[]gardenv1alpha1.ExpirableVersion `json:"versions"`
}

// ShootWorkerFlavor defines the worker pools that should be tested
type ShootWorkerFlavor struct {
	WorkerPools []gardenv1alpha1.Worker `json:"workerPools"`
}

// ExtendedConfiguration specifies extended configuration for shoot flavors that are deployed into a preexisting landscape
type ExtendedConfiguration struct {
	Cloudprofile  string `json:"cloudprofile"`
	ProjectName   string `json:"projectName"`
	SecretBinding string `json:"secretBinding"`
	Region        string `json:"region"`
	Zone          string `json:"zone"`

	FloatingPoolName     string `json:"floatingPoolName"`
	LoadbalancerProvider string `json:"loadbalancerProvider"`
}

// ExtendedShootConfiguration specifies extended configuration for shoots that are deployed into a preexisting landscape
type ExtendedShootConfiguration struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	ExtendedConfiguration
}
