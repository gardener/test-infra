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

package templates

import (
	"path"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

const (
	ConfigSeedName  = "SEED"
	ConfigSeedValue = "base"

	ConfigControlplaneProviderPathName   = "CONTROLPLANE_PROVIDER_CONFIG_FILEPATH"
	ConfigInfrastructureProviderPathName = "INFRASTRUCTURE_PROVIDER_CONFIG_FILEPATH"

	ConfigShootName                 = "SHOOT_NAME"
	ConfigShootAnnotations          = "SHOOT_ANNOTATIONS"
	ConfigProjectNamespaceName      = "PROJECT_NAMESPACE"
	ConfigK8sVersionName            = "K8S_VERSION"
	ConfigCloudproviderName         = "CLOUDPROVIDER"
	ConfigProviderTypeName          = "PROVIDER_TYPE"
	ConfigAllowPrivilegedContainers = "ALLOW_PRIVILEGED_CONTAINERS"
	ConfigCloudprofileName          = "CLOUDPROFILE"
	ConfigSecretBindingName         = "SECRET_BINDING"
	ConfigRegionName                = "REGION"
	ConfigZoneName                  = "ZONE"
)

var (
	ConfigControlplaneProviderPath   = path.Join(testmachinery.TM_SHARED_PATH, "generators/controlplane.yaml")
	ConfigInfrastructureProviderPath = path.Join(testmachinery.TM_SHARED_PATH, "generators/infra.yaml")
)

// CreateShootConfig describes the configuration for a create-shoot step
type CreateShootConfig struct {
	ShootName                 string
	ShootAnnotations          map[string]string
	Namespace                 string
	K8sVersion                string
	AllowPrivilegedContainers *bool
}

// GetStepCreateShoot generates the shoot creation step for a specific cloudprovider
// A shoot in a specific version is created depending on the gardener configuration whereas
// the default for commits is the new api.
func GetStepCreateShoot(_ GardenerConfig, cloudprovider common.CloudProvider, name string, dependencies []string, cfg *CreateShootConfig) ([]*v1beta1.DAGStep, string, error) {
	return stepCreateShootV1beta1(cloudprovider, name, dependencies, cfg)
}

func GetStepDeleteShoot(name, createShootStepName, shootName string, dependencies []string) v1beta1.DAGStep {
	return v1beta1.DAGStep{
		Name: name,
		Definition: v1beta1.StepDefinition{
			Name: "delete-shoot",
			Config: []v1beta1.ConfigElement{
				{
					Type:  v1beta1.ConfigTypeEnv,
					Name:  ConfigShootName,
					Value: shootName,
				},
			},
		},
		UseGlobalArtifacts: false,
		DependsOn:          dependencies,
		ArtifactsFrom:      createShootStepName,
		Annotations:        nil,
	}
}
