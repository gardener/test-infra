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

package util

import (
	"github.com/Masterminds/semver"
	gardenv1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/pkg/errors"
)

// ConvertStringToVersion converts a string to gardener experable versions
func ConvertStringToVersion(v string) gardenv1alpha1.ExpirableVersion {
	return gardenv1alpha1.ExpirableVersion{
		Version:        v,
		ExpirationDate: nil,
	}
}

// ConvertStringArrayToVersions converts a string array of versions to gardener experable versions
func ConvertStringArrayToVersions(versions []string) []gardenv1alpha1.ExpirableVersion {
	expVersions := make([]gardenv1alpha1.ExpirableVersion, len(versions))
	for i, v := range versions {
		expVersions[i] = ConvertStringToVersion(v)
	}
	return expVersions
}

// ContainsCloudprovider checks whether a cloudprovier is part of an array of cloudproviders
func ContainsCloudprovider(cloudproviders []common.CloudProvider, cloudprovider common.CloudProvider) bool {
	for _, cp := range cloudproviders {
		if cp == cloudprovider {
			return true
		}
	}
	return false
}

// GetLatestVersion returns the latest image from a list of expirable versions
func GetLatestVersion(rawVersions []gardenv1alpha1.ExpirableVersion) (gardenv1alpha1.ExpirableVersion, error) {
	if len(rawVersions) == 0 {
		return gardenv1alpha1.ExpirableVersion{}, errors.New("no kubernetes versions found")
	}

	var (
		latestExpVersion gardenv1alpha1.ExpirableVersion
		latestVersion    *semver.Version
	)

	for _, rawVersion := range rawVersions {
		v, err := semver.NewVersion(rawVersion.Version)
		if err != nil {
			return gardenv1alpha1.ExpirableVersion{}, err
		}
		if latestVersion == nil || v.GreaterThan(latestVersion) {
			latestVersion = v
			latestExpVersion = rawVersion
		}
	}
	return latestExpVersion, nil
}
