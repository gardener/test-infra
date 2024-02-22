// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package locations

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1/validation"
	"github.com/gardener/test-infra/pkg/testmachinery/locations/location"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

// NewTestLocations takes the parsed CRD Locations and fetches all TestDefintions from all locations.
func NewTestLocations(log logr.Logger, testLocations []tmv1beta1.TestLocation) (Locations, error) {
	testDefs := map[string]*testdefinition.TestDefinition{}

	if len(testLocations) == 0 {
		return nil, errors.New("no test locations defined")
	}

	for i, testLocation := range testLocations {
		t := testLocation
		locationLog := log.WithValues("testlocation", i)

		if err := validation.ValidateTestLocation(field.NewPath("spec", "testDefLocations").Index(i), t); len(err) != 0 {
			return nil, err.ToAggregate()
		}

		if testLocation.Type == tmv1beta1.LocationTypeGit {
			loc, err := location.NewGitLocation(locationLog, &t)
			if err != nil {
				locationLog.Error(err, "unable to create git testlocation")
				continue
			}
			err = loc.SetTestDefs(testDefs)
			if err != nil {
				locationLog.Info("unable to get testdefinitions", "error", err.Error())
				continue
			}
		}
		if testLocation.Type == tmv1beta1.LocationTypeLocal {
			loc := location.NewLocalLocation(locationLog, &t)
			err := loc.SetTestDefs(testDefs)
			if err != nil {
				locationLog.Info("unable to get testdefinitions", "error", err)
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
