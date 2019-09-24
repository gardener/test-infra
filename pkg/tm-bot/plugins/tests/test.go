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
	"github.com/gardener/gardener/pkg/client/kubernetes"
	_default "github.com/gardener/test-infra/pkg/testrunner/renderer/default"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins"
	"github.com/go-logr/logr"
	"time"
)

type test struct {
	runID string
	log   logr.Logger

	k8sClient kubernetes.Interface
	timeout   time.Duration
	interval  time.Duration

	config      _default.Config
	testLabel   string
	hibernation bool
	dryRun      bool
}

func New(log logr.Logger, k8sClient kubernetes.Interface) plugins.Plugin {
	return &test{
		log:       log.WithName("test"),
		k8sClient: k8sClient,
		timeout:   5 * time.Hour,
		interval:  1 * time.Minute,
	}
}

func (t *test) New(runID string) plugins.Plugin {
	return &test{
		runID:     runID,
		log:       t.log,
		k8sClient: t.k8sClient,
		timeout:   t.timeout,
		interval:  t.interval,
	}
}

func (t *test) Command() string {
	return "test"
}

func (t *test) Description() string {
	return "Runs a default gardener test with the specified flavors"
}

func (t *test) Example() string {
	return "/test "
}
