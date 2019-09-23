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
	"fmt"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

const (
	ConfigSeedName  = "SEED"
	ConfigSeedValue = "base"

	ConfigShootName            = "SHOOT_NAME"
	ConfigProjectNamespaceName = "PROJECT_NAMESPACE"
	ConfigK8sVersionName       = "K8S_VERSION"
	ConfigCloudproviderName    = "CLOUDPROVIDER"
	ConfigCloudprofileName     = "CLOUDPROFILE"
	ConfigSecretBindingName    = "SECRET_BINDING"
	ConfigRegionName           = "REGION"
	ConfigZoneName             = "ZONE"
)

// CreateShootConfig describes the configuration for a create-shoot step
type CreateShootConfig struct {
	ShootName  string
	Namespace  string
	K8sVersion string
}

// GetStepCreateShoot generates the shoot creation step for a specific cloudprovider
func GetStepCreateShoot(cloudprovider gardenv1beta1.CloudProvider, name string, dependencies []string, cfg *CreateShootConfig) (v1beta1.DAGStep, error) {
	stepConfig := defaultShootConfig(cfg)
	switch cloudprovider {
	case gardenv1beta1.CloudProviderAWS:
		if name == "" {
			name = "create-shoot-aws"
		}
		stepConfig = AWSShootConfig(stepConfig)
		break
	case gardenv1beta1.CloudProviderGCP:
		if name == "" {
			name = "create-shoot-gcp"
		}
		stepConfig = GCPShootConfig(stepConfig)
		break
	case gardenv1beta1.CloudProviderAzure:
		if name == "" {
			name = "create-shoot-azure"
		}
		stepConfig = AzureShootConfig(stepConfig)
		break
	default:
		return v1beta1.DAGStep{}, fmt.Errorf("unsupported cloudprovider %s", cloudprovider)
	}

	return v1beta1.DAGStep{
		Name: name,
		Definition: v1beta1.StepDefinition{
			Name:   "create-shoot",
			Config: stepConfig,
		},
		UseGlobalArtifacts: false,
		DependsOn:          dependencies,
		ArtifactsFrom:      "",
		Annotations:        nil,
	}, nil
}

func defaultShootConfig(cfg *CreateShootConfig) []v1beta1.ConfigElement {
	return []v1beta1.ConfigElement{
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigShootName,
			Value: cfg.ShootName,
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigProjectNamespaceName,
			Value: cfg.Namespace,
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigK8sVersionName,
			Value: cfg.K8sVersion,
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigSeedName,
			Value: ConfigSeedValue,
		},
	}
}

func GCPShootConfig(cfg []v1beta1.ConfigElement) []v1beta1.ConfigElement {
	return append(cfg, []v1beta1.ConfigElement{
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigCloudproviderName,
			Value: "gcp",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigCloudprofileName,
			Value: "gcp",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigSecretBindingName,
			Value: "core-gcp-gcp",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigRegionName,
			Value: "europe-west1",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigZoneName,
			Value: "europe-west1-b",
		},
	}...)
}

func AWSShootConfig(cfg []v1beta1.ConfigElement) []v1beta1.ConfigElement {
	return append(cfg, []v1beta1.ConfigElement{
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigCloudproviderName,
			Value: "aws",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigCloudprofileName,
			Value: "aws",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigSecretBindingName,
			Value: "core-aws-aws",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigRegionName,
			Value: "eu-west-1",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigZoneName,
			Value: "eu-west-1b",
		},
	}...)
}

func AzureShootConfig(cfg []v1beta1.ConfigElement) []v1beta1.ConfigElement {
	return append(cfg, []v1beta1.ConfigElement{
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigCloudproviderName,
			Value: "azure",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigCloudprofileName,
			Value: "azure",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigSecretBindingName,
			Value: "core-azure-azure",
		},
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  ConfigRegionName,
			Value: "westeurope",
		},
	}...)
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
