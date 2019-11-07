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
	"github.com/Masterminds/semver"
	gardenv1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// Validate validates a shoot flavor and checks if all necessary attributes are set
func Validate(identifier string, flavor *common.ShootFlavor) error {
	var allErrs *multierror.Error

	if flavor.Provider == "" {
		allErrs = multierror.Append(allErrs, fmt.Errorf("%s.provider: value has to be defined", identifier))
	}

	if flavor.KubernetesVersions.Versions == nil && flavor.KubernetesVersions.Pattern == nil {
		allErrs = multierror.Append(allErrs, fmt.Errorf("%s.kubernetes : Kubernetes versions or a pattern has to be defined", identifier))
	}

	if len(flavor.Workers) != 0 {
		for i, pool := range flavor.Workers {
			identifier := fmt.Sprintf("%s[%d].workerPools", identifier, i)
			if len(pool.WorkerPools) == 0 {
				allErrs = multierror.Append(allErrs, fmt.Errorf("%s: at least one worker pool has to be defined", identifier))
				continue
			}
			for j, workers := range pool.WorkerPools {
				if workers.Machine.Image == nil {
					allErrs = multierror.Append(allErrs, fmt.Errorf("%s[%d].machine.image: value has to be defined", identifier, j))
				}
			}
		}
	}

	return util.ReturnMultiError(allErrs)
}

// New creates an internal representation of raw shoot flavors.
// It also parses the flavors and creates the resulting shoots.
func New(rawFlavors []*common.ShootFlavor) (*Flavors, error) {
	versions := make(map[common.CloudProvider][]gardenv1alpha1.ExpirableVersion, 0)
	machineImages := make(map[common.CloudProvider][]gardenv1alpha1.MachineImage, 0)
	addVersion := addKubernetesVersionFunc(versions)
	addMachineImage := addMachineImagesFunc(machineImages)

	shoots := make([]*common.Shoot, 0)
	for i, rawFlavor := range rawFlavors {
		if err := Validate(fmt.Sprintf("flavor[%d]", i), rawFlavor); err != nil {
			return nil, err
		}

		versions, err := ParseKubernetesVersions(rawFlavor.KubernetesVersions)
		if err != nil {
			return nil, err
		}
		for _, k8sVersion := range versions {
			addVersion(rawFlavor.Provider, k8sVersion)

			if len(rawFlavor.Workers) != 0 {
				for _, workers := range rawFlavor.Workers {
					for _, pool := range workers.WorkerPools {
						addMachineImage(rawFlavor.Provider, pool.Machine.Image.Name, pool.Machine.Image.Version)
					}

					shoots = append(shoots, &common.Shoot{
						Provider:          rawFlavor.Provider,
						KubernetesVersion: k8sVersion,
						Workers:           workers.WorkerPools,
					})
				}
				continue
			}
			shoots = append(shoots, &common.Shoot{
				Provider:          rawFlavor.Provider,
				KubernetesVersion: k8sVersion,
			})
		}
	}

	return &Flavors{
		Info:                   rawFlavors,
		shoots:                 shoots,
		usedKubernetesVersions: versions,
		usedMachineImages:      machineImages,
	}, nil
}

// GetShoots returns a list of all shoots that are defined by the given flavors.
func (f *Flavors) GetShoots() []*common.Shoot {
	if f.shoots == nil {
		return []*common.Shoot{}
	}
	return f.shoots
}

// GetUsedKubernetesVersions returns a list of unique kubernetes versions used across all shoots.
func (f *Flavors) GetUsedKubernetesVersions() map[common.CloudProvider][]gardenv1alpha1.ExpirableVersion {
	if f.usedKubernetesVersions == nil {
		return make(map[common.CloudProvider][]gardenv1alpha1.ExpirableVersion, 0)
	}
	return f.usedKubernetesVersions
}

func (f *Flavors) GetUsedMachineImages() map[common.CloudProvider][]gardenv1alpha1.MachineImage {
	if f.usedMachineImages == nil {
		return make(map[common.CloudProvider][]gardenv1alpha1.MachineImage, 0)
	}
	return f.usedMachineImages
}

// addKubernetesVersionFunc adds a new kubernetes version to a list of unique versions per cloudprovider.
func addKubernetesVersionFunc(versions map[common.CloudProvider][]gardenv1alpha1.ExpirableVersion) func(common.CloudProvider, gardenv1alpha1.ExpirableVersion) {
	used := make(map[common.CloudProvider]map[string]interface{}, 0)
	return func(provider common.CloudProvider, version gardenv1alpha1.ExpirableVersion) {

		if _, ok := used[provider]; !ok {
			used[provider] = map[string]interface{}{version.Version: new(interface{})}
			versions[provider] = []gardenv1alpha1.ExpirableVersion{version}
			return
		}

		if _, ok := used[provider][version.Version]; !ok {
			used[provider][version.Version] = new(interface{})
			versions[provider] = append(versions[provider], version)
		}
	}
}

// addKubernetesVersionFunc adds a new kubernetes version to a list of unique versions per cloudprovider.
func addMachineImagesFunc(images map[common.CloudProvider][]gardenv1alpha1.MachineImage) func(common.CloudProvider, string, string) {
	used := make(map[common.CloudProvider]map[string]map[string]interface{}, 0)
	indexMapping := make(map[common.CloudProvider]map[string]int, 0)
	return func(provider common.CloudProvider, name, version string) {
		if _, ok := used[provider]; !ok {
			used[provider] = map[string]map[string]interface{}{
				name: {version: new(interface{})},
			}
			indexMapping[provider] = map[string]int{name: 0}

			images[provider] = []gardenv1alpha1.MachineImage{
				{
					Name:     name,
					Versions: []gardenv1alpha1.ExpirableVersion{{Version: version}},
				},
			}
			return
		}

		if _, ok := used[provider][name]; !ok {
			used[provider][name] = map[string]interface{}{version: new(interface{})}
			indexMapping[provider][name] = len(images[provider]) - 1

			images[provider] = append(images[provider], gardenv1alpha1.MachineImage{
				Name:     name,
				Versions: []gardenv1alpha1.ExpirableVersion{{Version: version}},
			})
			return
		}

		if _, ok := used[provider][name][version]; !ok {
			used[provider][name][version] = new(interface{})
			index := indexMapping[provider][name]
			images[provider][index].Versions = append(images[provider][index].Versions, gardenv1alpha1.ExpirableVersion{Version: version})
		}
	}
}

// ParseKubernetesVersions parses kubernetes versions flavor and returns a list of kubernetes versions.
// This function will not read from cloudprofile as it is meant to be used in the full gardener tests where there is no landscape
// to fetch versions at this point in time.
func ParseKubernetesVersions(versions common.ShootKubernetesVersionFlavor) ([]gardenv1alpha1.ExpirableVersion, error) {
	if versions.Versions != nil && len(*versions.Versions) != 0 {
		for _, v := range *versions.Versions {
			_, err := semver.NewVersion(v.Version)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid version %s", v)
			}
		}
		return *versions.Versions, nil
	}
	if *versions.Pattern != "" {
		return nil, errors.New("unable to read versions from cloudprofile. Only concrete kubernetes versions are allowed")
	}
	return nil, errors.New("no kubernetes versions or patterns are defined")
}
