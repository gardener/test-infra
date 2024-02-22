// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-multierror"
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
