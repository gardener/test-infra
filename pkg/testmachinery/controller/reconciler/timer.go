// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"fmt"
	"time"
)

func (r *TestmachineryReconciler) addTimer(key string, t time.Duration, f func()) error {
	r.Logger.V(5).Info("add timer", "duration", t.String(), "key", key)
	if t := r.timers[key]; t != nil {
		return fmt.Errorf("a timer is already defined for %s", key)
	}
	timer := time.NewTimer(t)
	go func() {
		<-timer.C
		delete(r.timers, key)
		f()
	}()
	r.timers[key] = timer
	return nil
}
