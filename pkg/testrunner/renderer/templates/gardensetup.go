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
	"encoding/base64"
	"encoding/json"
	"github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util/strconf"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

const (
	GardenCredentialsSecretName = "garden-test"
)

type GardenerConfig struct {
	Version string

	ImageTag string
	Commit   string
}

func GetStepCreateGardener(locationSet string, dependencies []string, baseClusterCloudprovider gardenv1beta1.CloudProvider, kubernetesVersions []string, cfg GardenerConfig) (v1beta1.DAGStep, error) {
	stepConfig, err := AppendGardenerConfig(GetCreateGardenerConfig(baseClusterCloudprovider), cfg)
	if err != nil {
		return v1beta1.DAGStep{}, err
	}
	stepConfig, err = AppendKubernetesVersionConfig(stepConfig, kubernetesVersions)
	if err != nil {
		return v1beta1.DAGStep{}, err
	}
	return v1beta1.DAGStep{
		Name: "create-garden",
		Definition: v1beta1.StepDefinition{
			Name:        "create-garden",
			LocationSet: &locationSet,
			Config:      stepConfig,
		},
		UseGlobalArtifacts: false,
		DependsOn:          dependencies,
	}, nil
}

func GetStepDeleteGardener(createGardenStep *v1beta1.DAGStep, locationSet string, dependencies []string, pause bool) v1beta1.DAGStep {
	return v1beta1.DAGStep{
		Name: "delete-garden",
		Definition: v1beta1.StepDefinition{
			Name:        "delete-garden",
			LocationSet: &locationSet,
			Config:      createGardenStep.Definition.Config,
		},
		UseGlobalArtifacts: false,
		ArtifactsFrom:      createGardenStep.Name,
		DependsOn:          dependencies,
		Pause: &v1beta1.Pause{
			Enabled:              pause,
			ResumeTimeoutSeconds: &common.DefaultPauseTimeout,
		},
	}
}

func AppendGardenerConfig(stepConfig []v1beta1.ConfigElement, cfg GardenerConfig) ([]v1beta1.ConfigElement, error) {
	if cfg.Version == "" && cfg.ImageTag == "" && cfg.Commit == "" {
		return stepConfig, nil
	}

	if cfg.Version != "" {
		return append(stepConfig, v1beta1.ConfigElement{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  "GARDENER_VERSION",
			Value: cfg.Version,
		}), nil
	}

	if cfg.ImageTag != "" && cfg.Commit == "" {
		return nil, errors.New("gardener commit has to be defined")
	}
	if cfg.ImageTag == "" && cfg.Commit != "" {
		return nil, errors.New("gardener image has to be defined")
	}

	return append(stepConfig, v1beta1.ConfigElement{
		Type:  v1beta1.ConfigTypeEnv,
		Name:  "GARDENER_IMAGE_TAG",
		Value: cfg.ImageTag,
	},
		v1beta1.ConfigElement{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  "GARDENER_COMMIT",
			Value: cfg.Commit,
		}), nil

}

func AppendKubernetesVersionConfig(stepConfig []v1beta1.ConfigElement, versions []string) ([]v1beta1.ConfigElement, error) {
	private := true
	kubernetesConstraint := v1alpha1.KubernetesSettings{
		Versions: make([]v1alpha1.ExpirableVersion, len(versions)),
	}
	for i, version := range versions {
		kubernetesConstraint.Versions[i] = v1alpha1.ExpirableVersion{
			Version: version,
		}
	}

	rawVersions, err := json.Marshal(kubernetesConstraint)
	if err != nil {
		return nil, err
	}
	b64Versions := base64.StdEncoding.EncodeToString(rawVersions)

	return append(stepConfig, v1beta1.ConfigElement{
		Type:    v1beta1.ConfigTypeFile,
		Name:    "K8S_VERSIONS",
		Path:    "/tm/gs/kubernetes_versions.json",
		Value:   b64Versions,
		Private: &private,
	}), nil
}

func GetCreateGardenerConfig(cloudprovider gardenv1beta1.CloudProvider) []v1beta1.ConfigElement {
	private := true
	return []v1beta1.ConfigElement{
		{
			Type:  v1beta1.ConfigTypeEnv,
			Name:  "BASE_CLOUDPROVIDER",
			Value: string(cloudprovider),
		},
		{
			Type:    v1beta1.ConfigTypeFile,
			Name:    "gcloud",
			Private: &private,
			Path:    "/tmp/garden/gcloud.json",
			ValueFrom: &strconf.ConfigSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: GardenCredentialsSecretName},
					Key:                  "gcloud.json",
				},
			},
		},
		{
			Type:    v1beta1.ConfigTypeEnv,
			Name:    "ACCESS_KEY_ID",
			Private: &private,
			ValueFrom: &strconf.ConfigSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: GardenCredentialsSecretName},
					Key:                  "accessKeyID",
				},
			},
		},
		{
			Type:    v1beta1.ConfigTypeEnv,
			Name:    "SECRET_ACCESS_KEY_ID",
			Private: &private,
			ValueFrom: &strconf.ConfigSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: GardenCredentialsSecretName},
					Key:                  "secretAccessKey",
				},
			},
		},
		{
			Type:    v1beta1.ConfigTypeEnv,
			Name:    "AZ_CLIENT_ID",
			Private: &private,
			ValueFrom: &strconf.ConfigSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: GardenCredentialsSecretName},
					Key:                  "clientID",
				},
			},
		},
		{
			Type:    v1beta1.ConfigTypeEnv,
			Name:    "AZ_CLIENT_SECRET",
			Private: &private,
			ValueFrom: &strconf.ConfigSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: GardenCredentialsSecretName},
					Key:                  "clientSecret",
				},
			},
		},
		{
			Type:    v1beta1.ConfigTypeEnv,
			Name:    "AZ_SUBSCRIPTION_ID",
			Private: &private,
			ValueFrom: &strconf.ConfigSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: GardenCredentialsSecretName},
					Key:                  "subscriptionID",
				},
			},
		},
		{
			Type:    v1beta1.ConfigTypeEnv,
			Name:    "AZ_TENANT_ID",
			Private: &private,
			ValueFrom: &strconf.ConfigSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: GardenCredentialsSecretName},
					Key:                  "tenantID",
				},
			},
		},
	}
}
