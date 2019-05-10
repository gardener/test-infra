package locations

import (
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
)

// Locations is an interface which provides functions for receiving TestDefinitions that are fetched from testDefLocations.
type Locations interface {
	GetTestDefinitions(*tmv1beta1.TestflowStep) ([]*testdefinition.TestDefinition, error)
}

type Sets struct {
	Sets    map[string]*Set
	Default *Set
}

type Set struct {
	Info         tmv1beta1.LocationSet
	TestLocation Locations
}

type testLocation struct {
	Info            []tmv1beta1.TestLocation
	TestDefinitions map[string]*testdefinition.TestDefinition
}
