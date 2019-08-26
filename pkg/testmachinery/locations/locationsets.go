package locations

import (
	"fmt"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

// NewLocations returns locations interface for a testrun
func NewLocations(log logr.Logger, spec tmv1beta1.TestrunSpec) (Locations, error) {
	if spec.LocationSets != nil {
		return NewSetLocations(log, spec.LocationSets)
	}

	if len(spec.TestLocations) > 0 {
		return NewTestLocations(log, spec.TestLocations)
	}

	return nil, errors.New("no test locations defined")
}

func NewSetLocations(log logr.Logger, sets []tmv1beta1.LocationSet) (Locations, error) {
	locSets := &Sets{
		Sets: make(map[string]*Set),
	}
	var firstSet *Set
	for _, set := range sets {
		testlocation, err := NewTestLocations(log, set.Locations)
		if err != nil {
			return nil, err
		}
		locSet := &Set{
			Info:         set,
			TestLocation: testlocation,
		}
		locSets.Sets[set.Name] = locSet

		if set.Default {
			locSets.Default = locSet
		}
		if firstSet == nil {
			firstSet = locSet
		}
	}
	if locSets.Default == nil {
		locSets.Default = firstSet
	}
	return locSets, nil
}

func (s *Sets) GetTestDefinitions(step tmv1beta1.StepDefinition) ([]*testdefinition.TestDefinition, error) {
	if step.LocationSet == nil {
		return s.Default.TestLocation.GetTestDefinitions(step)
	}

	if _, ok := s.Sets[*step.LocationSet]; !ok {
		return nil, fmt.Errorf("LocationSet %s is not defined in spec.locationSets", *step.LocationSet)
	}

	return s.Sets[*step.LocationSet].TestLocation.GetTestDefinitions(step)
}
