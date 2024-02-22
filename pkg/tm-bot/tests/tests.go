// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
)

var runs = &Runs{
	m:     sync.Mutex{},
	tests: make(map[string]*Run),
}

type Runs struct {
	watch watch.Watch
	m     sync.Mutex
	tests map[string]*Run
}

type Run struct {
	Testrun *v1beta1.Testrun
	Event   *github.GenericRequestEvent
}

func NewRuns(w watch.Watch) *Runs {
	r := Runs{
		watch: w,
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

// GetClient returns the controller runtime kubernetes client
func (r *Runs) GetClient() client.Client {
	return r.watch.Client()
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
	r.m.Lock()
	defer r.m.Unlock()

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

// SubTestConfig configures a specific test when called with the test command and
// the defined sub command
type TestConfig struct {
	// FilePath is the path to the testrun file that is executed.
	FilePath string `json:"testrunPath"`
	// Template configures the test to template the given file before execution.
	Template bool `json:"template"`
	// SetValues are the additional values that are used to template a testrun.
	SetValues []string
}
