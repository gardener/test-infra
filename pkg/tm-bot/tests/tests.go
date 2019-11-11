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

package tests

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/pkg/errors"
	"sync"
)

var runs = &Runs{
	m:     sync.Mutex{},
	tests: make(map[string]*Run),
}

type Runs struct {
	m     sync.Mutex
	tests map[string]*Run
}

type Run struct {
	Testrun *v1beta1.Testrun
	Event   *github.GenericRequestEvent
}

func NewRuns() *Runs {
	r := Runs{
		m:     sync.Mutex{},
		tests: make(map[string]*Run),
	}
	runs = &r
	return &r
}

// IsRunning indicates if a test is running for a Event (org, repo, pr)
func (r *Runs) IsRunning(event *github.GenericRequestEvent) bool {
	_, ok := r.tests[uniqueEventString(event)]
	return ok
}

// GetRunning returns the currently running Testrun for a Event (org, repo, pr)
func GetRunning(event *github.GenericRequestEvent) (*Run, bool) {
	return runs.GetRunning(event)
}

// GetRunning returns the currently running Testrun for a Event (org, repo, pr)
func (r *Runs) GetRunning(event *github.GenericRequestEvent) (*Run, bool) {
	run, ok := r.tests[uniqueEventString(event)]
	return run, ok
}

// GetAllRunning returns all running tests
func GetAllRunning() []*Run {
	runlist := make([]*Run, 0)
	for _, run := range runs.tests {
		runlist = append(runlist, run)
	}
	return runlist
}

func (r *Runs) Add(event *github.GenericRequestEvent, tr *v1beta1.Testrun) error {
	runs.m.Lock()
	defer runs.m.Unlock()

	if r.tests[uniqueEventString(event)] != nil {
		return errors.New("A test is already running for this PR.")
	}

	r.tests[uniqueEventString(event)] = &Run{
		Testrun: tr,
		Event:   event,
	}

	return nil
}

func (r *Runs) Remove(event *github.GenericRequestEvent) {
	r.m.Lock()
	defer r.m.Unlock()

	delete(r.tests, uniqueEventString(event))
}

func uniqueEventString(event *github.GenericRequestEvent) string {
	return fmt.Sprintf("%s/%s/%d", event.GetOwnerName(), event.GetRepositoryName(), event.Number)
}
