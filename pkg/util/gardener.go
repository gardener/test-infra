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
	"context"
	"fmt"
	"github.com/Masterminds/semver"
	gardenv1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
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

// GetCloudProfile returns the cloudprofile
func GetCloudProfile(k8sClient client.Client, profileName string) (gardenv1alpha1.CloudProfile, error) {
	var cloudprofile gardenv1alpha1.CloudProfile
	if err := k8sClient.Get(context.TODO(), client.ObjectKey{Name: profileName}, &cloudprofile); err != nil {
		return cloudprofile, err
	}
	return cloudprofile, nil
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

// FilterPatchVersions keeps only versions with newest patch versions. E.g. 1.15.1, 1.14.4, 1.14.3, will result in 1.15.1, 1.14.4
func FilterPatchVersions(cloudProfileVersions []gardenv1alpha1.ExpirableVersion) ([]gardenv1alpha1.ExpirableVersion, error) {
	type versionWrapper struct {
		expirableVersion gardenv1alpha1.ExpirableVersion
		semverVersion    *semver.Version
	}
	newestPatchVersionMap := make(map[string]versionWrapper)
	for _, rawVersion := range cloudProfileVersions {
		parsedVersion, err := semver.NewVersion(rawVersion.Version)
		if err != nil {
			return nil, err
		}
		majorMinor := fmt.Sprintf("%d.%d", parsedVersion.Major(), parsedVersion.Minor())
		if newestPatch, ok := newestPatchVersionMap[majorMinor]; !ok || newestPatch.semverVersion.LessThan(parsedVersion) {
			newestPatchVersionMap[majorMinor] = versionWrapper{
				expirableVersion: rawVersion,
				semverVersion:    parsedVersion,
			}
		}
	}

	newestPatchVersions := make([]gardenv1alpha1.ExpirableVersion, 0)
	for _, version := range newestPatchVersionMap {
		newestPatchVersions = append(newestPatchVersions, version.expirableVersion)
	}
	return newestPatchVersions, nil
}

// FilterExpiredVersions removes all expired versions from the list.
func FilterExpiredVersions(versions []gardenv1alpha1.ExpirableVersion) []gardenv1alpha1.ExpirableVersion {
	filtered := make([]gardenv1alpha1.ExpirableVersion, 0)
	for _, v := range versions {
		if v.ExpirationDate == nil || v.ExpirationDate.Time.After(time.Now()) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// DecMinorVersion decreases the minor version of 1 and sets the patch version to 0.
func DecMinorVersion(v *semver.Version) (*semver.Version, error) {
	minor := v.Minor() - 1
	if minor < 0 {
		minor = 0
	}
	vPrev := fmt.Sprintf("%d.%d.%d", v.Major(), minor, 0)
	return semver.NewVersion(vPrev)
}
