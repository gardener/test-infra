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

package locations

import (
	"errors"
	"fmt"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/locations/location"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	log "github.com/sirupsen/logrus"
)

// NewTestLocations takes the parsed CRD Locations and fetches all TestDefintions from all locations.
func NewTestLocations(testLocations []tmv1beta1.TestLocation) (Locations, error) {
	testDefs := map[string]*testdefinition.TestDefinition{}

	if len(testLocations) == 0 {
		return nil, errors.New("no TestDefinition locations defined")
	}

	for i, testLocation := range testLocations {
		t := testLocation

		if err := ValidateTestLocation(fmt.Sprintf("spec.testDefLocations.[%d]", i), t); err != nil {
			return nil, err
		}

		if testLocation.Type == tmv1beta1.LocationTypeGit {
			loc, err := location.NewGitLocation(&t)
			if err != nil {
				log.Warn(err.Error())
				continue
			}
			err = loc.SetTestDefs(testDefs)
			if err != nil {
				log.Warn(err.Error())
				continue
			}
		}
		if testLocation.Type == tmv1beta1.LocationTypeLocal {
			loc := location.NewLocalLocation(&t)
			err := loc.SetTestDefs(testDefs)
			if err != nil {
				log.Error(err.Error())
			}
		}
	}
	return &testLocation{testLocations, testDefs}, nil
}

// GetTestDefinitions returns all TestDefinitions of a StepDefinition with their location.GetTestDefinitions
// It errors if a TestDefinition cannot be found.
func (l *testLocation) GetTestDefinitions(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error) {
	if step.Name != "" {
		if l.TestDefinitions[step.Name] == nil {
			return nil, fmt.Errorf("TestDefinition %s cannot be found", step.Name)
		}
		td := l.TestDefinitions[step.Name].Copy()
		return []*testdefinition.TestDefinition{td}, nil
	}
	if step.Label != "" {
		defs := make([]*testdefinition.TestDefinition, 0)
		for _, td := range l.TestDefinitions {
			if td.HasLabel(step.Label) {
				newTd := td.Copy()
				defs = append(defs, newTd)
			}
		}
		return defs, nil
	}

	return nil, fmt.Errorf("unknown testrun step")
}
