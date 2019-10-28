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

package controller

import (
	"fmt"
	"time"
)

func (r *TestrunReconciler) addTimer(key string, t time.Duration, f func()) error {
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

func (r *TestrunReconciler) stopTimer(key string) error {
	if t := r.timers[key]; t == nil {
		return nil
	}
	r.timers[key].Stop()
	delete(r.timers, key)
	return nil
}
