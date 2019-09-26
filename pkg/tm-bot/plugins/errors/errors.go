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

import "fmt"

type PluginError struct {
	short string
	long  string
}

var _ error = &PluginError{}

func (e *PluginError) Error() string {
	return e.long
}

// New creates new plugin error.
// short error message is meant for the github output
// whereas the long message is for the internal log
func New(short, long string) error {
	return &PluginError{
		short: short,
		long:  long,
	}
}

// Wrapf wraps an error as long and adds an additional short question
func Wrapf(err error, shortMsg string, a ...interface{}) error {
	return New(fmt.Sprintf(shortMsg, a...), err.Error())
}

// Wrap wraps an error as long and adds an additional short question
func Wrap(err error, shortMsg string) error {
	return New(shortMsg, err.Error())
}

// ShortForError returns short message for the error
func ShortForError(err error) string {
	switch t := err.(type) {
	case *PluginError:
		return t.short
	}
	return "Unkown error"
}
