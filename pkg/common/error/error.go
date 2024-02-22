// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package errors

import (
	"github.com/pkg/errors"
)

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
