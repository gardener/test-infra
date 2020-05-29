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

package shootflavors

import (
	"fmt"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/hashicorp/go-multierror"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateExtendedFlavor validates extended a shoot flavors.
func ValidateExtendedFlavor(identifier string, flavor *common.ExtendedShootFlavor) error {
	var allErrs *multierror.Error

	if flavor.Provider == "" {
		allErrs = multierror.Append(allErrs, fmt.Errorf("%s.provider: value has to be defined", identifier))
	}
	if flavor.CloudprofileName == "" {
		allErrs = multierror.Append(allErrs, fmt.Errorf("%s.cloudprovider: value has to be defined", identifier))
	}
	if flavor.ProjectName == "" {
		allErrs = multierror.Append(allErrs, fmt.Errorf("%s.projectName: value has to be defined", identifier))
	}
	if flavor.SecretBinding == "" {
		allErrs = multierror.Append(allErrs, fmt.Errorf("%s.secretBinding: value has to be defined", identifier))
	}
	if flavor.Region == "" {
		allErrs = multierror.Append(allErrs, fmt.Errorf("%s.region: value has to be defined", identifier))
	}

	if flavor.KubernetesVersions.Versions == nil && flavor.KubernetesVersions.Pattern == nil {
		allErrs = multierror.Append(allErrs, fmt.Errorf("%s.kubernetes : Kubernetes versions or a pattern has to be defined", identifier))
	}

	if len(flavor.Workers) == 0 {
		return util.ReturnMultiError(multierror.Append(allErrs, fmt.Errorf("%s.workers: at least one worker flavor has to be defined", identifier)))
	}
	for i, pool := range flavor.Workers {
		if len(pool.WorkerPools) == 0 {
			allErrs = multierror.Append(allErrs, fmt.Errorf("%s.%d.workerPools: at least one worker pool has to be defined", identifier, i))
		}
	}

	return util.ReturnMultiError(allErrs)
}

// NewExtended creates an internal representation of raw extended shoot flavors.
// It also parses the flavors and creates the resulting extended shoots.
func NewExtended(k8sClient client.Client, rawFlavors []*common.ExtendedShootFlavor, shootPrefix string, filterPatchVersions bool) (*ExtendedFlavors, error) {
	versions := make(map[common.CloudProvider]gardencorev1beta1.KubernetesSettings, 0)
	addVersion := addKubernetesVersionFunc(versions)

	shoots := make([]*ExtendedFlavorInstance, 0)
	for i, rawFlavor := range rawFlavors {
		if err := ValidateExtendedFlavor(fmt.Sprintf("Flavors.%d", i), rawFlavor); err != nil {
			return nil, err
		}

		cloudprofile, err := util.GetCloudProfile(k8sClient, rawFlavor.CloudprofileName)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get cloudprofile %s", rawFlavor.CloudprofileName)
		}
		versions, err := util.GetK8sVersions(cloudprofile, rawFlavor.KubernetesVersions, filterPatchVersions)
		if err != nil {
			return nil, err
		}
		for _, k8sVersion := range versions {
			addVersion(rawFlavor.Provider, k8sVersion)

			for _, workers := range rawFlavor.Workers {
				pools, err := SetupWorker(cloudprofile, workers.WorkerPools)
				if err != nil {
					return nil, err
				}
				shoots = append(shoots, &ExtendedFlavorInstance{
					shoot: &common.ExtendedShoot{
						Shoot: common.Shoot{
							Description:               rawFlavor.Description,
							Provider:                  rawFlavor.Provider,
							KubernetesVersion:         k8sVersion,
							AllowPrivilegedContainers: rawFlavor.AllowPrivilegedContainers,
							Workers:                   pools,
						},
						ExtendedShootConfiguration: common.ExtendedShootConfiguration{
							Name:                  fmt.Sprintf("%s%s", shootPrefix, util.RandomString(3)),
							Namespace:             fmt.Sprintf("garden-%s", rawFlavor.ProjectName),
							Cloudprofile:          cloudprofile,
							ExtendedConfiguration: rawFlavor.ExtendedConfiguration,
						},
					},
				})
			}
		}
	}

	return &ExtendedFlavors{
		Info:                   rawFlavors,
		shoots:                 shoots,
		usedKubernetesVersions: versions,
	}, nil
}

// GetShoots returns a list of all shoots that are defined by the given flavors.
func (f *ExtendedFlavors) GetShoots() []*ExtendedFlavorInstance {
	if f.shoots == nil {
		return []*ExtendedFlavorInstance{}
	}
	return f.shoots
}

// GetUsedKubernetesVersions returns a list of unique kubernetes versions used across all shoots.
func (f *ExtendedFlavors) GetUsedKubernetesVersions() map[common.CloudProvider]gardencorev1beta1.KubernetesSettings {
	if f.usedKubernetesVersions == nil {
		return map[common.CloudProvider]gardencorev1beta1.KubernetesSettings{}
	}
	return f.usedKubernetesVersions
}

func NewExtendedFlavorInstance(shoot *common.ExtendedShoot) *ExtendedFlavorInstance {
	return &ExtendedFlavorInstance{shoot: shoot}
}

// New creates a new unique ExtendedFlavor shoot instance
func (i *ExtendedFlavorInstance) New() *common.ExtendedShoot {
	newFlavor := *i.shoot
	newFlavor.Name = fmt.Sprintf("%s-%s", newFlavor.Name, util.RandomString(3))
	return &newFlavor
}

func (i *ExtendedFlavorInstance) Get() *common.ExtendedShoot {
	return i.shoot
}
