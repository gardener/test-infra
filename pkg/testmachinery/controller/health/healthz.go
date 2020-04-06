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

package health

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"net/http"
	"sync"
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
