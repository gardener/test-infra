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

package testrunner

import (
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"time"
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
