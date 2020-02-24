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

package testrunner

import (
	"container/list"
	"fmt"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sync"
	"time"
)

// ExecutorConfig configures the execution order of a execution
type ExecutorConfig struct {
	// Serial describes of the items should be executed in serial
	Serial bool

	// BackoffPeriod is the duration to wait between the creation of a bucket of testruns
	// 0 means that all functions are started in parallel
	BackoffPeriod time.Duration

	// BackoffBucket is the number of parallel created testruns per backoff period
	// 0 disables the backoff
	BackoffBucket int
}

// Executor runs a set of functions in a preconfigured order
type Executor interface {
	AddItem(func())
	Run()
}

type executor struct {
	mut    sync.Mutex
	log    logr.Logger
	config ExecutorConfig
	items  *list.List
}

// NewExecutor creates a new function executor
func NewExecutor(log logr.Logger, config ExecutorConfig) (Executor, error) {
	if config.BackoffBucket < 0 {
		return nil, errors.New("a backoff bucket cannot be less than 0")
	}
	if config.BackoffPeriod < 0 {
		return nil, errors.New("a backoff period cannot be less than 0")
	}
	return &executor{
		mut:    sync.Mutex{},
		log:    log.WithName("executor"),
		config: config,
		items:  list.New(),
	}, nil
}

// pop returns the next element in the queue (first element of the list)
// and removes the element from the list
func (e *executor) pop() (func(), bool) {
	e.mut.Lock()
	defer e.mut.Unlock()
	elem := e.items.Front()
	e.items.Remove(elem)
	f, ok := elem.Value.(func())
	return f, ok
}

// len returns the thread safe length of the current queue
func (e *executor) len() int {
	e.mut.Lock()
	defer e.mut.Unlock()
	return e.items.Len()
}

// AddItem adds a item to the execution queue
func (e *executor) AddItem(f func()) {
	e.mut.Lock()
	defer e.mut.Unlock()
	e.items.PushBack(f)
	return
}

// Run executes all added items in the configured order
func (e *executor) Run() {
	var wg = &util.AdvancedWaitGroup{}

	var i = 0
	for e.len() != 0 {
		wg.Add(1)

		f, ok := e.pop()
		if !ok {
			e.log.V(3).Info("unable to cast queue element to func")
			continue
		}

		go func(i int, f func()) {
			defer wg.Done()

			// wait initial backoff before deploying the testrun
			if e.config.BackoffBucket > 0 {
				d := time.Duration(1)
				if !e.config.Serial {
					d = time.Duration(i / e.config.BackoffBucket)
				}
				time.Sleep(e.config.BackoffPeriod * d)
			}

			e.log.V(10).Info(fmt.Sprintf("running execution %d", i))
			f()
		}(i, f)

		// wait for the current run to finish if
		// - the runs should be executed in serial and item is the last of the current bucket
		// - or the function is the last element of the queue
		if e.config.Serial && util.IsLastElementOfBucket(i, e.config.BackoffBucket) {
			wg.Wait()
		}
		if e.len() == 0 {
			e.waitForLastElement(wg)
		}
		i++
	}
	// not needed but to be sure that all goroutines are finished
	wg.Wait()
}

// waitForLastElement waits until the last element has finished.
// The wait is canceled if another element is added to the queue and has to be processed.
func (e *executor) waitForLastElement(wg *util.AdvancedWaitGroup) {
	wg.WaitWithCancelFunc(func() bool {
		return e.len() > 0
	})
}
