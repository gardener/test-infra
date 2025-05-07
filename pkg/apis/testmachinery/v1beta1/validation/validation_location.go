package validation

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

// ValidateLocations validates testlocations.
func ValidateLocations(fldPath *field.Path, spec tmv1beta1.TestrunSpec) field.ErrorList {
	var allErrs field.ErrorList
	if spec.LocationSets == nil && len(spec.TestLocations) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("locationSets"), "no location for TestDefinitions defined"))
		return allErrs
	}

	if spec.LocationSets != nil {
		allErrs = append(allErrs, ValidateLocationSets(fldPath.Child("locationSets"), spec.LocationSets)...)
	}

	if len(spec.TestLocations) > 0 {
		allErrs = append(allErrs, ValidateTestLocations(fldPath.Child("testLocations"), spec.TestLocations)...)
	}

	return allErrs
}

// ValidateLocationSets validates a list of locationsets.
func ValidateLocationSets(fldPath *field.Path, sets []tmv1beta1.LocationSet) field.ErrorList {
	var allErrs field.ErrorList
	for i, set := range sets {
		allErrs = append(allErrs, ValidateLocationSet(fldPath.Index(i), set)...)
	}
	return allErrs
}

// ValidateLocationSet validates a location set.
func ValidateLocationSet(fldPath *field.Path, set tmv1beta1.LocationSet) field.ErrorList {
	var allErrs field.ErrorList
	if set.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "name has to be defined for locationSets"))
	}
	if len(set.Locations) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("locations"), "locations has to be defined for locationSets"))
	}
	allErrs = append(allErrs, ValidateTestLocations(fldPath.Child("locations"), set.Locations)...)
	return allErrs
}

// ValidateTestLocations validates the deprecated test locations.
func ValidateTestLocations(fldPath *field.Path, l []tmv1beta1.TestLocation) field.ErrorList {
	var allErrs field.ErrorList
	for i, loc := range l {
		allErrs = append(allErrs, ValidateTestLocation(fldPath.Index(i), loc)...)
	}
	return allErrs
}

// ValidateTestLocation validates a test location of a testrun
func ValidateTestLocation(fldPath *field.Path, l tmv1beta1.TestLocation) field.ErrorList {
	var allErrs field.ErrorList
	if l.Type == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("type"), "must be defined"))
		return allErrs
	}
	if l.Type != tmv1beta1.LocationTypeGit && l.Type != tmv1beta1.LocationTypeLocal {
		allErrs = append(allErrs, field.Invalid(
			fldPath.Child("type"),
			l.Type,
			fmt.Sprintf("Unknown TestDefinition location type. Supported types: %q, %q", tmv1beta1.LocationTypeGit, tmv1beta1.LocationTypeLocal)))
		return allErrs
	}
	switch l.Type {
	case tmv1beta1.LocationTypeGit:
		if l.Repo == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("repo"), "repo has to be defined for git TestDefinition locations"))
		}
		if l.Revision == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("revision"), "revision has to be defined for git TestDefinition locations"))
		}
	case tmv1beta1.LocationTypeLocal:
		if !testmachinery.IsRunInsecure() {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("type"), l.Type, "Local testDefinition locations are only available in insecure mode"))
			return allErrs
		}
		if l.HostPath == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("hostPath"), "hostPath has to be defined for local TestDefinition locations"))
		}
	}
	return allErrs
}
