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
	"github.com/gardener/test-infra/pkg/common"
	_default "github.com/gardener/test-infra/pkg/testrun_renderer/default"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
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

	config             _default.Config
	kubernetesVersions []string
	cloudproviders     []common.CloudProvider
	testLabel          string
	hibernation        bool
	dryRun             bool
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

func (_ *test) Authorization() github.AuthorizationType {
	return github.AuthorizationOrg
}

func (t *test) Description() string {
	return "Runs a default gardener test with the specified flavors"
}

func (t *test) Example() string {
	return "/test "
}

func (t *test) Config() string {
	return `
test:
  hostprovider: gardener # gardener or gke
  baseClusterCloudprovider: gcp # Cloudprovider of the selected host. Only applicable for hostprovider gardener

  gardensetup:
    revision: # StringOrGitHubConfig

  gardener:
    version: # StringOrGitHubConfig
    commit: # StringOrGitHubConfig

  shootFlavors:
  - cloudprovider: # cloudprovider e.g. aws
    kubernetesVersions: 
    - version: "" # expirable version
    workers:
    - workerPools: # gardener worker definitions
      - name: "wp1" 

# StringOrGitHubConfig
parameter: "string"

parameter:
  value: "string" # raw string value. Same as defining only a string
  path: test/path # read the file in the default branch of the repo (repo root will used to define the path) and return its content as a string
  prHead: true # use the commit sha of the current PR's head
`
}
