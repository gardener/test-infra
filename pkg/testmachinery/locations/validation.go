package locations

import (
	"fmt"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

func ValidateLocations(identifier string, spec tmv1beta1.TestrunSpec) error {
	if spec.LocationSets != nil {
		return ValidateLocationSets(fmt.Sprintf("%s.locationSets", identifier), spec.LocationSets)
	}

	if len(spec.TestLocations) > 0 {
		return ValidateTestLocations(fmt.Sprintf("%s.testLocations", identifier), spec.TestLocations)
	}

	return errors.New("no location for TestDefinitions defined")
}

func ValidateLocationSets(identifier string, sets []tmv1beta1.LocationSet) error {
	var result *multierror.Error
	for i, set := range sets {
		if err := ValidateLocationSet(fmt.Sprintf("%s.%d", identifier, i), set); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result.ErrorOrNil()
}

func ValidateLocationSet(identifier string, set tmv1beta1.LocationSet) error {
	if set.Name == "" {
		return fmt.Errorf("%s.name: Required value: name has to be defined for locationSets", identifier)
	}
	if set.Locations == nil || len(set.Locations) == 0 {
		return fmt.Errorf("%s.locations: Required value: locations has to be defined for locationSets", identifier)
	}
	return ValidateTestLocations(fmt.Sprintf("%s.locations", identifier), set.Locations)
}

func ValidateTestLocations(identifier string, l []tmv1beta1.TestLocation) error {
	for i, loc := range l {
		if err := ValidateTestLocation(fmt.Sprintf("%s.%d", identifier, i), loc); err != nil {
			return err
		}
	}
	return nil
}

// ValidateTestLocation validates a TestDefinition LocationSet of a testrun
func ValidateTestLocation(identifier string, l tmv1beta1.TestLocation) error {
	if l.Type == tmv1beta1.LocationTypeGit {
		if l.Repo == "" {
			return fmt.Errorf("%s.repo: Required value: repo has to be defined for git TestDefinition locations", identifier)
		}
		if l.Revision == "" {
			return fmt.Errorf("%s.revision: Required value: revision has to be defined for git TestDefinition locations", identifier)
		}
		return nil
	}
	if l.Type == tmv1beta1.LocationTypeLocal {
		if !testmachinery.IsRunInsecure() {
			return errors.New("Local testDefinition locations are only available in insecure mode")
		}
		if l.HostPath == "" {
			return fmt.Errorf("%s.hostPath: Required value: hostPath has to be defined for local TestDefinition locations", identifier)
		}
		return nil
	}
	if l.Type == "" {
		return fmt.Errorf("%s.type: Required value: type has to be defined for spec.testDefLocations", identifier)
	}
	return fmt.Errorf("%s.type: Unknown TestDefinition location type %s", identifier, l.Type)
}
