// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/hashicorp/go-multierror"
)

var (
	mutex      sync.Mutex
	conditions = map[string]Condition{}
)

// Condition defines a interface to check a specific health condition
type Condition interface {
	CheckHealth(ctx context.Context) error
}

func AddHealthCondition(name string, condition Condition) {
	mutex.Lock()
	conditions[name] = condition
	mutex.Unlock()
}

func Healthz() func(req *http.Request) error {
	return func(req *http.Request) error {
		var (
			ctx     = context.Background()
			allErrs *multierror.Error
		)
		defer ctx.Done()
		mutex.Lock()
		for name, condition := range conditions {
			if err := condition.CheckHealth(ctx); err != nil {
				allErrs = multierror.Append(allErrs, fmt.Errorf("%s: %s", name, err.Error()))
			}
		}
		mutex.Unlock()
		return allErrs.ErrorOrNil()
	}
}
