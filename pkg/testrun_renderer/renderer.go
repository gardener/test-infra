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
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/pkg/errors"
	"strings"
)

// AddBOMLocationsToTestrun adds component descriptor repositories as location to location sets
func AddBOMLocationsToTestrun(tr *tmv1beta1.Testrun, locationSetName string, components []*componentdescriptor.Component, useAsDefault bool) error {
	if tr == nil || len(components) == 0 {
		return nil
	}

	bomLocations := make([]tmv1beta1.TestLocation, 0)
	for _, component := range components {
		bomLocations = append(bomLocations, tmv1beta1.TestLocation{
			Type:     tmv1beta1.LocationTypeGit,
			Repo:     fmt.Sprintf("https://%s", component.Name),
			Revision: GetRevisionFromVersion(component.Version),
		})
	}

	// check if the locationSet already exists
	for i, set := range tr.Spec.LocationSets {
		if set.Name == locationSetName {
			set.Locations = append(bomLocations, set.Locations...)
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
			Locations: append(bomLocations, existingLocations...),
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
