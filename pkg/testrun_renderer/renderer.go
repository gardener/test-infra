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

package testrun_renderer

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
)

// AddLocationsToTestrun adds component descriptor repositories and the given additionalLocations as locations to location sets for the given Testrun tr.
func AddLocationsToTestrun(tr *tmv1beta1.Testrun, locationSetName string, components []*componentdescriptor.Component, useAsDefault bool, additionalLocations []common.AdditionalLocation) error {
	if tr == nil || len(components) == 0 {
		return nil
	}
	locations := make([]tmv1beta1.TestLocation, 0)
	for _, component := range components {
		var found bool
		repo := fmt.Sprintf("https://%s", component.Name)
		revision := GetRevisionFromVersion(component.Version)

		for i, l := range locations {
			if l.Repo == repo {
				found = true
				if revision == "master" || revision == "main" {
					locations[i].Revision = revision
				} else {
					existingVersion, err := semver.NewVersion(l.Revision)
					if err != nil {
						logger.Log.V(3).Info("Location's Duplicate Repo check for: %s: Failed to parse %s into a semVer compatible format. Only revision %s will be kept. Consider using additionalLocations to overwrite. Error: %s", repo, l.Revision, l.Revision, err.Error())
						break
					}
					incomingVersion, err := semver.NewVersion(revision)
					if err != nil {
						logger.Log.V(3).Info("Location's Duplicate Repo check for: %s: Failed to parse %s into a semVer compatible format. Only revision %s will be kept. Consider using additionalLocations to overwrite. Error: %s", repo, revision, l.Revision, err.Error())
						break
					}
					if incomingVersion.GreaterThan(existingVersion) {
						locations[i].Revision = revision
					}
				}
				break
			}
		}

		if !found {
			locations = append(locations, tmv1beta1.TestLocation{
				Type:     tmv1beta1.LocationTypeGit,
				Repo:     repo,
				Revision: revision,
			})
		}
	}

	for _, location := range additionalLocations {
		locationType, err := tmv1beta1.GetLocationType(location.Type)
		if err != nil {
			return err
		}
		locations = append(locations, tmv1beta1.TestLocation{
			Type:     locationType,
			Repo:     location.Repo,
			Revision: location.Revision,
		})
	}

	// check if the locationSet already exists
	for i, set := range tr.Spec.LocationSets {
		if set.Name == locationSetName {
			set.Locations = append(locations, set.Locations...)
			tr.Spec.LocationSets[i] = set
			tr.Spec.TestLocations = nil
			return nil
		}
		if useAsDefault && set.Default {
			return errors.New("a default location is already defined")
		}
	}

	// if old locations exist we migrate them to the new locationSet form
	existingLocations := tr.Spec.TestLocations
	tr.Spec.LocationSets = append(tr.Spec.LocationSets,
		tmv1beta1.LocationSet{
			Name:      locationSetName,
			Default:   useAsDefault,
			Locations: append(locations, existingLocations...),
		},
	)
	tr.Spec.TestLocations = nil

	return nil
}

// GetRevisionFromVersion parses the version of a component and returns its revision if applicable.
func GetRevisionFromVersion(version string) string {
	if strings.Contains(version, "dev") {
		splitVersion := strings.Split(version, "-")
		return splitVersion[len(splitVersion)-1]
	}
	return version
}
