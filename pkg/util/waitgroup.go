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

package util

import (
	"sync"
	"time"
)

// AdvancedWaitGroup implements the same interface as sync.WaitGroup.
// In addition a wait can be canceled during runtime when using the WaitWithCancel function
type AdvancedWaitGroup struct {
	noCopy

	mut   sync.Mutex
	count int
}

// Add adds delta to the wait counter.
// delta may be negative but the counter cannot be less then 0.
func (wg *AdvancedWaitGroup) Add(delta int) {
	wg.mut.Lock()
	defer wg.mut.Unlock()
	wg.count = wg.count + delta
	if wg.count < 0 {
		wg.count = 0
	}
}

// Done removes one element from the wait counter
func (wg *AdvancedWaitGroup) Done() {
	wg.Add(-1)
}

// Wait waits until the counter is 0
func (wg *AdvancedWaitGroup) Wait() {
	wg.WaitWithCancelFunc(func() bool {
		return false
	})
}

// WaitWithCancelFunc waits until the wait group count is zero or the cancel function returns true
func (wg *AdvancedWaitGroup) WaitWithCancelFunc(cancel func() bool) {
	for {
		if wg.count == 0 || cancel() {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

// noCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
