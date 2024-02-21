// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tm_bot

import (
	"net/http"
	"sync"

	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/pkg/version"
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
