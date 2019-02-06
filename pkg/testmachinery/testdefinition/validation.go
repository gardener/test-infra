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

package testdefinition

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/gardener/test-infra/pkg/testmachinery"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"

	apimachineryvalidation "k8s.io/apimachinery/pkg/util/validation"
)

// Validate validates a testdefinition.
func Validate(identifier string, td *tmv1beta1.TestDefinition) error {
	if td.Metadata.Name == "" {
		return fmt.Errorf("Invalid TestDefinition (%s): metadata.name : Required value: name has to be defined", identifier)
	}

	if err := validateName(identifier, td.Metadata.Name); err != nil {
		return err
	}

	if len(td.Spec.Command) == 0 {
		return fmt.Errorf("Invalid TestDefinition (%s) Name: \"%s\": spec.command : Required value: command has to be defined", identifier, td.Metadata.Name)
	}
	if td.Spec.Owner == "" || !isEmailValid(td.Spec.Owner) {
		return fmt.Errorf("Invalid TestDefinition (%s) Owner: \"%s\": spec.owner : Required value: valid email has to be defined", identifier, td.Spec.Owner)
	}
	if td.Spec.RecipientsOnFailure != "" && !isEmailListValid(td.Spec.RecipientsOnFailure) {
		return fmt.Errorf("Invalid TestDefinition (%s) ReceipientsOnFailure: \"%s\": spec.notifyOnFailure : Required value: valid email has to be defined", identifier, td.Spec.RecipientsOnFailure)
	}
	return nil
}

// validates the TestDefinition name. Therefore Kubernetes naming conventions and elasticsearch naming is considered.
// es conform:
// must not contain the #, \, /, *, ?, ", <, >, |, ,
// must not start with _, - or +
// must no be . or ..
// must be lowercase
func validateName(identifier, name string) error {
	if strings.Contains(name, ".") {
		return fmt.Errorf("Invalid TestDefinition (%s): metadata.name : Invalid value: name must not contain '.'", identifier)
	}

	// IsDNS1123Subdomain: lower case alphanumeric characters, '-' or '.', and must start and end with an alphanumeric character
	// used for e.g. statefulset names
	errMsgs := []string{}
	for _, msg := range apimachineryvalidation.IsDNS1123Subdomain(name) {
		errMsgs = append(errMsgs, msg)
	}

	if len(errMsgs) != 0 {
		return fmt.Errorf("Invalid TestDefinition (%s): metadata.name : Invalid value: %s", identifier, strings.Join(errMsgs, ";"))
	}

	return nil
}

func isEmailListValid(emailList string) bool {
	for _, email := range strings.Split(emailList, ",") {
		email = strings.Trim(email, " ")
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

// ValidateLocation validates a TestDefinition Location of a testrun
func ValidateLocation(identifier string, l *tmv1beta1.TestLocation) error {
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
