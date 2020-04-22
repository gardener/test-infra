// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package tm_bot

import (
	"github.com/gardener/test-infra/pkg/version"
	"github.com/go-logr/logr"
	"net/http"
	"sync"
)

var (
	mutex   sync.Mutex
	healthy = false
)

func UpdateHealth(isHealthy bool) {
	mutex.Lock()
	healthy = isHealthy
	mutex.Unlock()
}

func healthz(log logr.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		isHealthy := healthy
		mutex.Unlock()

		if isHealthy {
			_, err := w.Write([]byte(version.Get().String()))
			if err != nil {
				log.V(5).Info(err.Error())
				w.WriteHeader(http.StatusOK)
			}
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
