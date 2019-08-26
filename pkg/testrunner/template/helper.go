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
	"context"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/gardener/gardener/pkg/utils"
	"github.com/pkg/errors"

	"github.com/Masterminds/semver"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// getK8sVersions returns all K8s version that should be rendered by the chart
func getK8sVersions(parameters *ShootTestrunParameters) ([]string, error) {
	if !parameters.MakeVersionMatrix && parameters.K8sVersion != "" {
		return []string{parameters.K8sVersion}, nil
	}
	// if the kubernetes version is not set, get the latest version defined by the cloudprofile
	if !parameters.MakeVersionMatrix && parameters.K8sVersion == "" {
		version, err := getLatestK8sVersion(parameters.GardenKubeconfigPath, parameters.Cloudprofile, parameters.Cloudprovider)
		if err != nil {
			return nil, fmt.Errorf("'k8s-version' is not defined nor can it be read from the cloudprofile: %s", err.Error())
		}
		return []string{version}, nil
	}
	if parameters.MakeVersionMatrix {
		return getK8sVersionsFromCloudprofile(parameters.GardenKubeconfigPath, parameters.Cloudprofile, parameters.Cloudprovider)
	}

	return nil, fmt.Errorf("no K8s version can be specified")
}

func getK8sVersionsFromCloudprofile(gardenerKubeconfigPath, cloudprofile, cloudprovider string) ([]string, error) {
	ctx := context.Background()
	defer ctx.Done()
	k8sGardenClient, err := kubernetes.NewClientFromFile("", gardenerKubeconfigPath, client.Options{
		Scheme: kubernetes.GardenScheme,
	})
	if err != nil {
		return nil, err
	}

	profile := &gardenv1beta1.CloudProfile{}
	err = k8sGardenClient.Client().Get(ctx, types.NamespacedName{Name: cloudprofile}, profile)
	if err != nil {
		return nil, err
	}
	cloudProfileVersions, err := getCloudproviderVersions(profile, cloudprovider)
	if err != nil {
		return nil, err
	}
	if len(cloudProfileVersions) == 0 {
		return nil, fmt.Errorf("no kubernetes versions found for cloudprofile %s", cloudprofile)
	}
	return filterPatchVersions(cloudProfileVersions)
}

// filterPatchVersions keeps only versions with newest patch versions. E.g. 1.15.1, 1.14.4, 1.14.3, will result in 1.15.1, 1.14.4
func filterPatchVersions(cloudProfileVersions []string) ([]string, error) {
	newestPatchVersionMap := make(map[string]*semver.Version)
	for _, rawVersion := range cloudProfileVersions {
		parsedVersion, err := semver.NewVersion(rawVersion)
		if err != nil {
			return nil, err
		}
		majorMinor := fmt.Sprintf("%d.%d", parsedVersion.Major(), parsedVersion.Minor())
		if newestPatch, ok := newestPatchVersionMap[majorMinor]; !ok || newestPatch.LessThan(parsedVersion) {
			newestPatchVersionMap[majorMinor] = parsedVersion
		}
	}

	newestPatchVersions := make([]string, 0)
	for _, version := range newestPatchVersionMap {
		newestPatchVersions = append(newestPatchVersions, version.String())
	}
	return newestPatchVersions, nil
}

func getLatestK8sVersion(gardenerKubeconfigPath, cloudprofile, cloudprovider string) (string, error) {
	rawVersions, err := getK8sVersionsFromCloudprofile(gardenerKubeconfigPath, cloudprofile, cloudprovider)
	if err != nil {
		return "", err
	}

	if len(rawVersions) == 0 {
		return "", fmt.Errorf("no kubernetes versions found for cloudprofle %s", cloudprofile)
	}

	versions := make([]*semver.Version, len(rawVersions))
	for i, rawVersion := range rawVersions {
		v, err := semver.NewVersion(rawVersion)
		if err == nil {
			versions[i] = v
		}
	}
	sort.Sort(semver.Collection(versions))

	return versions[len(versions)-1].String(), nil
}

func getCloudproviderVersions(profile *gardenv1beta1.CloudProfile, cloudprovider string) ([]string, error) {

	switch gardenv1beta1.CloudProvider(cloudprovider) {
	case gardenv1beta1.CloudProviderAWS:
		return profile.Spec.AWS.Constraints.Kubernetes.Versions, nil
	case gardenv1beta1.CloudProviderGCP:
		return profile.Spec.GCP.Constraints.Kubernetes.Versions, nil
	case gardenv1beta1.CloudProviderAzure:
		return profile.Spec.Azure.Constraints.Kubernetes.Versions, nil
	case gardenv1beta1.CloudProviderOpenStack:
		return profile.Spec.OpenStack.Constraints.Kubernetes.Versions, nil
	case gardenv1beta1.CloudProviderAlicloud:
		return profile.Spec.Alicloud.Constraints.Kubernetes.Versions, nil
	default:
		return nil, fmt.Errorf("Unsupported cloudprovider %s", cloudprovider)
	}
}

func addBOMLocationsToTestrun(tr *tmv1beta1.Testrun, locationSetName string, components []*componentdescriptor.Component) {
	if tr == nil || components == nil {
		return
	}

	bomLocations := make([]tmv1beta1.TestLocation, 0)
	for _, component := range components {
		bomLocations = append(bomLocations, tmv1beta1.TestLocation{
			Type:     tmv1beta1.LocationTypeGit,
			Repo:     fmt.Sprintf("https://%s", component.Name),
			Revision: getRevisionFromVersion(component.Version),
		})
	}

	// check if the locationSet already exists
	for i, set := range tr.Spec.LocationSets {
		if set.Name == locationSetName {
			set.Locations = append(bomLocations, set.Locations...)
			tr.Spec.LocationSets[i] = set
			tr.Spec.TestLocations = nil
			return
		}
	}

	// if old locations exist we migrate them to the new locationSet form
	if len(tr.Spec.TestLocations) == 0 {
		return
	}
	existingLocations := tr.Spec.TestLocations
	tr.Spec.LocationSets = []tmv1beta1.LocationSet{
		{
			Name:      locationSetName,
			Locations: append(bomLocations, existingLocations...),
		},
	}
	tr.Spec.TestLocations = nil
}

// getRevisionFromVersion parses the version of a component and returns its revision if applicable.
func getRevisionFromVersion(version string) string {
	if strings.Contains(version, "dev") {
		splitVersion := strings.Split(version, "-")
		return splitVersion[len(splitVersion)-1]
	}
	return version
}

func addAnnotationsToTestrun(tr *tmv1beta1.Testrun, annotations map[string]string) {
	if tr == nil {
		return
	}
	tr.Annotations = utils.MergeStringMaps(tr.Annotations, annotations)
}

func getGardenerVersionFromComponentDescriptor(componentDescriptor componentdescriptor.ComponentList) string {
	for _, component := range componentDescriptor {
		if component == nil {
			continue
		}
		if component.Name == "github.com/gardener/gardener" {
			return component.Version
		}
	}
	return ""
}

func readFileValues(files []string) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	for _, file := range files {
		var newValues map[string]interface{}
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to read file %s", file)
		}
		if err := yaml.Unmarshal(data, &newValues); err != nil {
			return nil, errors.Wrapf(err, "unable to unmarshal yaml file %s", file)
		}
		values = utils.MergeMaps(values, newValues)
	}
	return values, nil
}
