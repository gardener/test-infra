// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testrunner

import (
	"time"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
)

// Config are configuration of the environment like the testmachinery cluster or S3 store
// where the testrunner executes the testrun.
type Config struct {
	// Testrun watch controller
	Watch watch.Watch

	// Namespace where the testrun is deployed.
	Namespace string

	// Max wait time for a testrun to finish.
	Timeout time.Duration

	// Number of testrun retries after a failed run
	FlakeAttempts int

	// NoExecutionGroup configures if a execution group id should be injected into every testrun.
	NoExecutionGroup bool

	// ExecutionGroupID is injected into every testrun if explicitly given, else ExecutionGroupID gets generated on runtime
	ExecutionGroupID string

	ExecutorConfig
}

// RunEventFunc is called every time a new testrun is triggered
// Also notifies for retries
type RunEventFunc func(run *Run)

// Run describes a testrun that is executed by the testrunner.
// It consists of a testrun and its metadata
type Run struct {
	// Specify internal info for specific run types
	Info     interface{}
	Testrun  *tmv1beta1.Testrun
	Metadata *metadata.Metadata
	Error    error

	Rerenderer Rerenderer
}

// Rerenderer is instance that rerenders the current run to make it retryable.
type Rerenderer interface {
	Rerender(tr *tmv1beta1.Testrun) (*Run, error)
}

// RunList represents a list of Runs.
type RunList []*Run
