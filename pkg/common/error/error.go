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

package errors

import "github.com/pkg/errors"

// error reasons
type CommonReason string

const (
	// error is not clear
	ReasonUnknown = ""

	// timeout error
	ReasonTimeout = "Timeout"

	// unable to create the resource
	ReasonNotCreated = "NotCreated"

	// resource could not be rendered error
	ReasonNotRendered = "NotRendered"

	// resource could not be found
	ReasonNotFound = "NotFound"

	// request is of wrong or unexpected type
	ReasonWrongType = "WrongType"
)

type commonError struct {
	message string
	reason  CommonReason
}

var _ error = &commonError{}

// Error implements the error interface
func (e *commonError) Error() string {
	return e.message
}

// newError returns a new common error with a reason
func newError(reason CommonReason, message string) error {
	return &commonError{
		message: message,
		reason:  reason,
	}
}

// NewTimeoutError returns an error indicating that a resource occurred
// during execution of a testrun.
func NewTimeoutError(message string) error {
	return newError(ReasonTimeout, message)
}

// NewNotCreatedError returns an error indicating that a resource could not be created
func NewNotCreatedError(message string) error {
	return newError(ReasonNotCreated, message)
}

// NewNotCreatedError returns an error indicating that a resource could not be created
func NewNotRenderedError(message string) error {
	return newError(ReasonNotRendered, message)
}

// NewNotFoundError returns an error indicating that a resource could not be found
func NewNotFoundError(message string) error {
	return newError(ReasonNotFound, message)
}

// NewWrongTypeError returns an error indicating that a request was of a wrong or unexpected type
func NewWrongTypeError(message string) error {
	return newError(ReasonWrongType, message)
}

// IsTimeout determines if the error indicates a timeout
func IsTimeout(err error) bool {
	return reasonForError(err) == ReasonTimeout
}

// IsNotCreated determines if the error indicates
// that the resource was not created
func IsNotCreated(err error) bool {
	return reasonForError(err) == ReasonNotCreated
}

// IsNotRendered determines if the error indicates
// that the resource was not rendered
func IsNotRendered(err error) bool {
	return reasonForError(err) == ReasonNotCreated
}

// IsNotFound determines if the error indicates
// that the resource was not found
func IsNotFound(err error) bool {
	return reasonForError(err) == ReasonNotFound
}

// IsWrongType determines if the error indicates
// that the resource request was of a wrong or unexpected type
func IsWrongType(err error) bool {
	return reasonForError(err) == ReasonWrongType
}

// reasonForError returns the testrunner CommonReason for a particular error.
func reasonForError(err error) CommonReason {
	switch t := errors.Cause(err).(type) {
	case *commonError:
		return t.reason
	}
	return ReasonUnknown
}
