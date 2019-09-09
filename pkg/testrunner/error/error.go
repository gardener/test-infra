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

// testrunner specific error reasons
type TestrunnerReason string

const (
	// testrunner error is not clear
	TestrunnerReasonUnknown = ""

	// testrunner ran into a timeout
	TestrunnerReasonTimeout = "Timeout"

	// testrunner was unable to create the testrun
	TestrunnerReasonNotCreated = "NotCreated"

	// testrunner was unable to render any testruns
	TestrunnerReasonNotRendered = "NotRendered"
)

type TestrunnerError struct {
	message string
	reason  TestrunnerReason
}

var _ error = &TestrunnerError{}

// Error implements the error interface
func (e *TestrunnerError) Error() string {
	return e.message
}

// New returns a new testrunner error
func New(reason TestrunnerReason, message string) error {
	return &TestrunnerError{
		message: message,
		reason:  reason,
	}
}

// NewTimeoutError returns an error indicating that a timeout occurred
// during execution of a testrun.
func NewTimeoutError(message string) error {
	return New(TestrunnerReasonTimeout, message)
}

// NewNotCreatedError returns an error indicating that a testrun could not be created
func NewNotCreatedError(message string) error {
	return New(TestrunnerReasonNotCreated, message)
}

// NewNotCreatedError returns an error indicating that a testrun could not be created
func NewNotRenderedError(message string) error {
	return New(TestrunnerReasonNotRendered, message)
}

// IsTimeout determines if the error indicates a timeout
func IsTimeout(err error) bool {
	return reasonForError(err) == TestrunnerReasonTimeout
}

// IsNotCreated indicates if the error indicates that
// a testrun could not be created.
func IsNotCreated(err error) bool {
	return reasonForError(err) == TestrunnerReasonNotCreated
}

// IsNotRendered indicates if the error indicates that
// no testrun could be rendered.
func IsNotRendered(err error) bool {
	return reasonForError(err) == TestrunnerReasonNotCreated
}

// reasonForError returns the testrunner reason for a particular error.
func reasonForError(err error) TestrunnerReason {
	switch t := err.(type) {
	case *TestrunnerError:
		return t.reason
	}
	return TestrunnerReasonUnknown
}
