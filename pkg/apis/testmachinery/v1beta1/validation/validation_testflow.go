// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
)

// ValidateTestFlow validates the structure of a testflow.
func ValidateTestFlow(fldPath *field.Path, testflow tmv1beta1.TestFlow) field.ErrorList {
	var allErrs field.ErrorList
	usedStepNames := make(map[string]*tmv1beta1.DAGStep)

	for i, step := range testflow {
		stepPath := fldPath.Index(i)

		if step.ArtifactsFrom != "" && step.UseGlobalArtifacts {
			allErrs = append(allErrs, field.Forbidden(stepPath.Child("useGlobalArtifacts"), "useGlobalArtifacts is not allowed when '.artifactsFrom' is defined"))
		}

		if step.Name == "" {
			allErrs = append(allErrs, field.Required(stepPath.Child("name"), "must not be emtpy"))
		}

		if _, ok := usedStepNames[step.Name]; ok {
			allErrs = append(allErrs, field.Duplicate(stepPath.Child("name"), step.Name))
		} else {
			usedStepNames[step.Name] = step
		}

		if err := ValidateStep(stepPath, step.Definition); err != nil {
			return err
		}
	}

	valHelper := newTFValidationHelper(usedStepNames)
	for i, step := range testflow {
		stepPath := fldPath.Index(i)

		// validate that depended steps exist
		for _, dependsOn := range step.DependsOn {
			if found, _ := valHelper.HasDependentStep(dependsOn, step); !found {
				allErrs = append(allErrs, field.NotFound(stepPath.Child("dependsOn"), dependsOn))
			}
		}
		// validate dependency cycle
		if found, path := valHelper.HasDependentStep(step.Name, step); found {
			allErrs = append(allErrs, field.Forbidden(stepPath.Child("dependsOn"), fmt.Sprintf("dependency cycle detected: %s", path)))
		}

		// validate artifact from step exists
		if step.ArtifactsFrom != "" {
			if _, ok := usedStepNames[step.ArtifactsFrom]; !ok {
				allErrs = append(allErrs, field.NotFound(stepPath.Child("artifactsFrom"), step.ArtifactsFrom))
				continue
			}
			// check if the artifact from is defined by a parent step
			if found, _ := valHelper.HasDependentStep(step.ArtifactsFrom, step); !found {
				allErrs = append(allErrs, field.Forbidden(stepPath.Child("artifactsFrom"), "artifact source step is not a preceding step"))
			}
		}
	}
	return allErrs
}

// ValidateStep validates a step definition
func ValidateStep(fldPath *field.Path, definition tmv1beta1.StepDefinition) field.ErrorList {
	var allErrs field.ErrorList
	if definition.Label == "" && definition.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name/label"), "name or label have to be specified"))
	}
	if definition.Name != "" {
		allErrs = append(allErrs, ValidateName(fldPath.Child("name"), definition.Name)...)
	}

	if definition.Condition != tmv1beta1.ConditionTypeAlways &&
		definition.Condition != tmv1beta1.ConditionTypeSuccess &&
		definition.Condition != tmv1beta1.ConditionTypeError &&
		definition.Condition != "" {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("condition"), definition.Condition, "invalid condition type"))
	}

	allErrs = append(allErrs, ValidateConfigList(fldPath.Child("config"), definition.Config)...)
	return allErrs
}

type testflowValidationHelper struct {
	stepNameToStep    map[string]*tmv1beta1.DAGStep
	dependentStepName string
	alreadyChecked    sets.Set[string]
}

func newTFValidationHelper(stepNameToStep map[string]*tmv1beta1.DAGStep) *testflowValidationHelper {
	return &testflowValidationHelper{
		stepNameToStep: stepNameToStep,
		alreadyChecked: sets.New[string](),
	}
}

// HasDependentStep validates if the dependentStepName is a parent of the step.
// This function is also used to detect dependency cycles when the dependentStepName == step.Name.
// Returns if the dependent step was found and the optional used path.
func (v *testflowValidationHelper) HasDependentStep(dependentStepName string, step *tmv1beta1.DAGStep) (bool, string) {
	v.dependentStepName = dependentStepName
	defer func() {
		v.alreadyChecked = sets.New[string]()
		v.dependentStepName = ""
	}()
	return v.checkPreviousStepHasDependentStep(field.NewPath(v.dependentStepName), v.dependentStepName, step.DependsOn)
}

func (v *testflowValidationHelper) checkPreviousStepHasDependentStep(path *field.Path, stepName string, parents []string) (bool, string) {
	if v.alreadyChecked.Has(stepName) {
		return false, ""
	}
	v.alreadyChecked.Insert(stepName)
	for _, parent := range parents {
		pPath := path.Child(parent)
		parentStep, ok := v.stepNameToStep[parent]
		if !ok {
			return false, pPath.String()
		}
		if parent == v.dependentStepName {
			return true, pPath.String()
		}

		if len(parentStep.DependsOn) == 0 {
			continue
		}
		if util.StringArrayContains(parentStep.DependsOn, v.dependentStepName) {
			return true, pPath.String()
		}
		if ok, path := v.checkPreviousStepHasDependentStep(pPath, parent, parentStep.DependsOn); ok {
			return ok, path
		}
	}
	return false, ""
}
