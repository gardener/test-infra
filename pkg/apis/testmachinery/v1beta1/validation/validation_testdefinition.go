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

package validation

import (
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"

	apimachineryvalidation "k8s.io/apimachinery/pkg/util/validation"
)

// ValidateTestDefinition validates a testdefinition.
func ValidateTestDefinition(fldPath *field.Path, td *tmv1beta1.TestDefinition) field.ErrorList {
	var allErrs field.ErrorList
	if td.GetName() == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "must be defined"))
	} else {
		allErrs = append(allErrs, ValidateName(fldPath.Child("name"), td.GetName())...)
	}

	specPath := fldPath.Child("spec")
	if len(td.Spec.Command) == 0 {
		allErrs = append(allErrs, field.Required(specPath.Child("command"), "must be defined"))
	}
	if td.Spec.Owner == "" {
		allErrs = append(allErrs, field.Required(specPath.Child("owner"), "must be defined"))
	} else if !isEmailValid(td.Spec.Owner) {
		allErrs = append(allErrs, field.Invalid(specPath.Child("owner"), td.Spec.Owner, "must be a valid email address"))
	}
	if len(td.Spec.RecipientsOnFailure) != 0 && !isEmailListValid(td.Spec.RecipientsOnFailure) {
		allErrs = append(allErrs, field.Invalid(specPath.Child("recipientsOnFailure"), td.Spec.RecipientsOnFailure, "must be a list of valid email addresses"))
	}

	for i, label := range td.Spec.Labels {
		labelPath := specPath.Child("labels").Index(i)
		allErrs = append(allErrs, ValidateLabelName(labelPath, label)...)
	}
	return allErrs
}

// ValidateName validates the TestDefinition name. Therefore Kubernetes naming conventions and elasticsearch naming is considered.
// es conform:
// must not contain the #, \, /, *, ?, ", <, >, |, ,
// must not start with _, - or +
// must no be . or ..
// must be lowercase
func ValidateName(fldPath *field.Path, name string) field.ErrorList {
	var allErrs field.ErrorList
	if strings.Contains(name, ".") {
		allErrs = append(allErrs, field.Invalid(fldPath, name, "name must not contain '.'"))
	}

	// IsDNS1123Subdomain: lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character
	// used for e.g. statefulset names
	errMsgs := append([]string{}, apimachineryvalidation.IsDNS1123Subdomain(name)...)

	if len(errMsgs) != 0 {
		allErrs = append(allErrs, field.Invalid(fldPath, name, strings.Join(errMsgs, ";")))
	}

	return allErrs
}

// ValidateLabelName validates the TestDefinition label string.
// label starting with "!" are not valid as this is considered to mean exclude label
func ValidateLabelName(fldPath *field.Path, label string) field.ErrorList {
	var allErrs field.ErrorList
	if strings.HasPrefix(label, "!") {
		allErrs = append(allErrs, field.Invalid(fldPath, label, "name must not contain '!'"))
	}
	return allErrs
}

func isEmailListValid(emailList []string) bool {
	for _, email := range emailList {
		if !isEmailValid(email) {
			return false
		}
	}
	return true
}

func isEmailValid(email string) bool {
	re := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return re.MatchString(email)
}
