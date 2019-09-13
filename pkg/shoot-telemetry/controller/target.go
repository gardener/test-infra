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
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/sample"
	log "github.com/sirupsen/logrus"
)

type target struct {
	endpoint  string
	transport *http.Transport
	active    bool
	archived  bool
	username  string
	password  string

	provider string
	seedName string

	series []*sample.Sample
}

func (c *controller) addTarget(shoot *gardenv1beta1.Shoot) {
	if c.filterShoot(shoot) {
		return
	}
	var key = common.GetShootKey(shoot)

	// Check if a target for the Shoot alrady exists
	t, exists := c.targets[key]
	if !exists {
		// Fetch the Shoot kubeconfig secret to configure the probe.
		secret, err := c.secrets.Lister().Secrets(shoot.ObjectMeta.Namespace).Get(fmt.Sprintf("%s.kubeconfig", shoot.ObjectMeta.Name))
		if err != nil {
			return
		}

		endpoint, err := c.determineShootInternalEndpoint(shoot)
		if err != nil {
			log.Debug(err.Error())
			return
		}

		seedName := "unknown"
		if shoot.Spec.Cloud.Seed != nil {
			seedName = *shoot.Spec.Cloud.Seed
		}

		t = &target{
			seedName: seedName,
			provider: common.DetermineShootProvider(shoot),
			endpoint: endpoint,
			username: string(secret.Data["username"]),
			password: string(secret.Data["password"]),
			transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}

		c.targetsMutex.Lock()
		c.targets[key] = t
		c.targetsMutex.Unlock()
		return
	}
	c.targetsMutex.Lock()
	t.archived = false
	c.targetsMutex.Unlock()
}

func (c *controller) removeTarget(shoot *gardenv1beta1.Shoot) {
	var key = common.GetShootKey(shoot)
	c.targetsMutex.Lock()
	if t, exists := c.targets[key]; exists {
		t.archived = true
	}
	c.targetsMutex.Unlock()
}

func (c *controller) observeTarget(t *target, stopCh <-chan struct{}) {
	var client = &http.Client{Transport: t.transport, Timeout: time.Millisecond * time.Duration(common.RequestTimeOut)}
	t.active = true

	common.Waiter(func() {
		if t.archived {
			t.active = false
			return
		}
		req, err := http.NewRequest(http.MethodGet, t.endpoint, nil)
		if err != nil {
			return
		}
		req.SetBasicAuth(t.username, t.password)

		startTime := time.Now()
		response, err := client.Do(req)
		if err != nil {
			log.Debugf("Request failed. Endpoint: %s", t.endpoint)
			t.series = append(t.series, sample.NewSample(0, startTime))
			return
		}
		response.Body.Close()
		t.series = append(t.series, sample.NewSample(response.StatusCode, startTime))
	}, c.config.CheckInterval, false, stopCh)
}

// filterShoot returns if a shoot should be watched based on the controller configuration
func (c *controller) filterShoot(shoot *gardenv1beta1.Shoot) bool {
	if c.config.ShootName != "" {
		if shoot.Name != c.config.ShootName || shoot.Namespace != c.config.ShootNamespace {
			return true
		}
	}
	if c.config.ShootNamespace != "" {
		if shoot.Namespace != c.config.ShootNamespace {
			return true
		}
	}
	return false
}
