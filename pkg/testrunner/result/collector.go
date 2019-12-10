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

package result

import (
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/logger"
	common2 "github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/testrunner"
	telemetryctrl "github.com/gardener/test-infra/pkg/testrunner/telemetry"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"os"
	"path"
	"time"
)

func New(log logr.Logger, config Config) (*Collector, error) {
	collector := &Collector{
		log:    log,
		config: config,
	}
	if config.EnableTelemetry {
		var err error
		collector.telemetry, err = telemetryctrl.New(logger.Log.WithName("telemetry-controller"), 1*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "unable to initialize telemetry controller")
		}
	}

	return collector, nil
}

func (c *Collector) PreRunShoots(kubeconfigPath string, runs testrunner.RunList) error {
	if c.telemetry == nil {
		return nil
	}
	if len(runs) == 0 {
		c.log.V(3).Info("no shoots registered")
		c.telemetry = nil
		return nil
	}

	shootsToWatch := make([]string, 0)
	for _, run := range runs {
		// check if run is a shoot run
		switch s := run.Info.(type) {
		case *common.ExtendedShoot:
			shootsToWatch = append(shootsToWatch, common2.GetShootKey(s.Name, s.Namespace))
			c.log.V(5).Info("registered shoot for telemetry watch", "name", s.Name, "namespace", s.Namespace)
		}
	}

	telemetryOutputDir := path.Join(c.config.OutputDir, "telemetry")
	if err := os.MkdirAll(telemetryOutputDir, os.ModePerm); err != nil {
		return err
	}
	if _, err := c.telemetry.StartForShoots(kubeconfigPath, telemetryOutputDir, shootsToWatch); err != nil {
		return errors.Wrap(err, "unable to start telemetry controller")
	}
	c.log.V(3).Info("registered shoots for telemetry measurement", "num", len(shootsToWatch))
	return nil
}

func (c *Collector) PreRunGardener(kubeconfigPath string) error {
	if c.telemetry == nil {
		return nil
	}
	telemetryOutputDir := path.Join(c.config.OutputDir, "telemetry")
	if err := os.MkdirAll(telemetryOutputDir, os.ModePerm); err != nil {
		return err
	}
	if _, err := c.telemetry.Start(kubeconfigPath, telemetryOutputDir); err != nil {
		return errors.Wrap(err, "unable to start telemetry controller")
	}
	return nil
}
