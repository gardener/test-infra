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

import (
	"fmt"
)

type PluginError struct {
	ShortMsg string
	LongMsg  string
	// recoverable indicates that the state of a plugin should not be deleted.
	// So it can be resumed after a restart.
	recoverable bool
	// showLong shows the LongMsg message when responded to the github message.
	// The LongMsg message will be printed in the detailed response if true.
	showLong bool
}

var _ error = &PluginError{}

func (e *PluginError) Error() string {
	return e.LongMsg
}

// New creates new plugin error.
// ShortMsg error message is meant for the github output
// whereas the LongMsg message is for the internal log
func New(short, long string) error {
	return &PluginError{
		ShortMsg: short,
		LongMsg:  long,
	}
}

func Builder() *PluginError {
	return &PluginError{}
}

// WithShort sets the ShortMsg message of a plugin error.
func (e *PluginError) WithShort(msg string) *PluginError {
	e.ShortMsg = msg
	return e
}

// WithShortf templates the given message and sets it as the ShortMsg message of a plugin error.
func (e *PluginError) WithShortf(format string, args ...interface{}) *PluginError {
	e.ShortMsg = fmt.Sprintf(format, args...)
	return e
}

// WithLong sets the LongMsg message of a plugin error.
func (e *PluginError) WithLong(msg string) *PluginError {
	e.LongMsg = msg
	return e
}

// WithLongf templates the given message and sets it as the LongMsg message of a plugin error.
func (e *PluginError) WithLongf(format string, args ...interface{}) *PluginError {
	e.LongMsg = fmt.Sprintf(format, args...)
	return e
}

// Recoverable sets the recoverability of the error to true.
func (e *PluginError) Recoverable() *PluginError {
	e.recoverable = true
	return e
}

// ShowLong shows the LongMsg message in github response.
func (e *PluginError) ShowLong() *PluginError {
	e.showLong = true
	return e
}

// NewRecoverable creates new plugin error that does not delete the state so it can be resumed after a restart.
// ShortMsg error message is meant for the github output
// whereas the LongMsg message is for the internal log
func NewRecoverable(short, long string) error {
	return &PluginError{
		recoverable: true,
		ShortMsg:    short,
		LongMsg:     long,
	}
}

// Wrapf wraps an error as LongMsg and adds an additional ShortMsg question
func Wrapf(err error, shortMsg string, a ...interface{}) error {
	return New(fmt.Sprintf(shortMsg, a...), err.Error())
}

// Wrap wraps an error as LongMsg and adds an additional ShortMsg question
func Wrap(err error, shortMsg string) error {
	return New(shortMsg, err.Error())
}

// IsRecoverable indicates if the error is recoverable
func IsRecoverable(err error) bool {
	switch t := err.(type) {
	case *PluginError:
		return t.recoverable
	}
	return false
}

// OmitLongMessage indicates if the error's LongMsg message should be
// omitted in the github response.
func OmitLongMessage(err error) bool {
	switch t := err.(type) {
	case *PluginError:
		return !t.showLong
	}
	return true
}

// ShortForError returns ShortMsg message for the error
func ShortForError(err error) string {
	switch t := err.(type) {
	case *PluginError:
		return t.ShortMsg
	}
	return "Unknown error"
}

// LongForError returns LongMsg message for the error
func LongForError(err error) string {
	switch t := err.(type) {
	case *PluginError:
		return t.LongMsg
	}
	return "Unknown error"
}
