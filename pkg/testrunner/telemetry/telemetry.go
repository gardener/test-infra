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

package telemetry

import (
	"github.com/gardener/test-infra/pkg/shoot-telemetry/analyse"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/config"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/controller"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"os"
	"path"
	"syscall"
	"time"
)

type Telemetry struct {
	log logr.Logger

	interval time.Duration
	err      error
	stopCh   chan struct{}
	signalCh chan os.Signal

	RawResultsPath string
}

func New(log logr.Logger, interval time.Duration) (*Telemetry, error) {
	return &Telemetry{
		log:      log,
		interval: interval,
	}, nil
}

// Start starts the telemetry measurement with a specific kubeconfig to watch all shoots
func (c *Telemetry) Start(kubeconfigPath, resultDir string) (string, error) {
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return "", err
	}
	c.RawResultsPath = path.Join(resultDir, "results.csv")
	cfg := &config.Config{
		KubeConfigPath: kubeconfigPath,
		CheckInterval:  c.interval,
		OutputDir:      resultDir,
		OutputFile:     c.RawResultsPath,
		DisableAnalyse: true,
	}

	c.StartWithConfig(cfg)

	return c.RawResultsPath, nil
}

// StartForShoot starts the telemetry measurement with a kubeconfig for a specific shoot
func (c *Telemetry) StartForShoot(shootName, shootNamespace, kubeconfigPath, resultDir string) (string, error) {
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return "", err
	}

	c.RawResultsPath = path.Join(resultDir, "results.csv")
	cfg := &config.Config{
		KubeConfigPath: kubeconfigPath,
		CheckInterval:  c.interval,
		OutputDir:      resultDir,
		OutputFile:     c.RawResultsPath,
		DisableAnalyse: true,
		ShootsFilter: map[string]bool{
			common.GetShootKey(shootName, shootNamespace): true,
		},
	}

	c.StartWithConfig(cfg)

	return c.RawResultsPath, nil
}

// StartForShoots starts the telemetry measurement with a kubeconfig for specific shoots
func (c *Telemetry) StartForShoots(kubeconfigPath, resultDir string, shootKeys []string) (string, error) {
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return "", err
	}

	c.RawResultsPath = path.Join(resultDir, "results.csv")
	cfg := &config.Config{
		KubeConfigPath: kubeconfigPath,
		CheckInterval:  c.interval,
		OutputDir:      resultDir,
		OutputFile:     c.RawResultsPath,
		DisableAnalyse: true,
		ShootsFilter:   make(map[string]bool, len(shootKeys)),
	}

	for _, key := range shootKeys {
		cfg.ShootsFilter[key] = true
	}

	c.StartWithConfig(cfg)

	return c.RawResultsPath, nil
}

func (c *Telemetry) StartWithConfig(cfg *config.Config) {
	c.stopCh = make(chan struct{})
	c.signalCh = make(chan os.Signal, 2)

	go func() {
		defer close(c.stopCh)
		if err := controller.StartController(cfg, c.signalCh); err != nil {
			c.err = err
			return
		}
	}()
}

// StopAndAnalyze stops the telemetry measurement and generates a result summary
func (c *Telemetry) StopAndAnalyze(resultDir, format string) (string, map[string]*analyse.Figures, error) {
	if err := c.Stop(); err != nil {
		return "", nil, err
	}
	return c.Analyze(resultDir, format)
}

// Stop stops the measurement of the telemetry controller
func (c *Telemetry) Stop() error {
	defer close(c.signalCh)
	c.log.V(3).Info("stop telemetry controller")
	if c.err != nil {
		return errors.Wrapf(c.err, "error during telemetry controller execution")
	}
	c.signalCh <- syscall.SIGTERM

	// wait for controller to finish
	<-c.stopCh
	return nil
}

// Analyze analyzes the previously measured values and returns the path to the summary
func (c *Telemetry) Analyze(resultDir, format string) (string, map[string]*analyse.Figures, error) {
	c.log.V(3).Info("analyze telemetry metrics")
	summaryOutput := ""
	if resultDir != "" {
		summaryOutput = path.Join(resultDir, "summary.json")
	}

	figures, err := analyse.Analyse(c.RawResultsPath, summaryOutput, format)
	if err != nil {
		return "", nil, errors.Wrap(err, "unable to analyze measurement")
	}
	return summaryOutput, figures, nil
}
