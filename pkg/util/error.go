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

package util

import (
	"fmt"
	"github.com/hashicorp/go-multierror"
	"reflect"
)

// ReturnMultiError takes an err object and returns a multierror with a custom format.
func ReturnMultiError(err error) error {
	if err == nil || reflect.ValueOf(err).IsNil() {
		return nil
	}

	if errs, ok := err.(*multierror.Error); ok {
		errs.ErrorFormat = func(errs []error) string {
			if len(errs) == 1 {
				return fmt.Sprintf("1 error occurred: %s", errs[0].Error())
			}

			errStr := fmt.Sprintf("%d errors occurred", len(errs))
			for _, err := range errs {
				errStr = fmt.Sprintf("%s - %s", errStr, err.Error())
			}
			return errStr
		}
		return errs.ErrorOrNil()
	}
	return err
}
