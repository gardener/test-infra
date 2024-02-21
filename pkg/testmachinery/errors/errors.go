// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var RecoverableErrors = []field.ErrorType{ErrorTestDefinitionRetrieveError}

const ErrorTestDefinitionRetrieveError field.ErrorType = "TestDefinitionRetrieveError"

// TestDefinitionRetrieveError creates a new TestDefinition retrieve error.
func TestDefinitionRetrieveError(fld *field.Path, detail string) *field.Error {
	return &field.Error{
		Type:     ErrorTestDefinitionRetrieveError,
		Field:    fld.String(),
		BadValue: nil,
		Detail:   detail,
	}
}
