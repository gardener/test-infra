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
	"os"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/analyse"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/config"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/controller"
)

type Telemetry struct {
	log logr.Logger
	mut sync.Mutex

	shootsFilter map[string]bool
	interval     time.Duration
	err          error
	started      bool
	stopCh       chan struct{}
	signalCh     chan os.Signal
}

func New(log logr.Logger, interval time.Duration) (*Telemetry, error) {
	return &Telemetry{
		log:          log,
		mut:          sync.Mutex{},
		interval:     interval,
		shootsFilter: make(map[string]bool),
	}, nil
}

// Start starts the telemetry measurement with a specific kubeconfig to watch all shoots
func (c *Telemetry) Start(kubeconfigPath, resultDir string) error {
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return err
	}
	if _, err := os.Stat(resultDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(resultDir, os.ModePerm); err != nil {
			return err
		}
	}

	cfg := &config.Config{
		KubeConfigPath: kubeconfigPath,
		CheckInterval:  c.interval,
		OutputDir:      resultDir,
		DisableAnalyse: true,
	}

	c.StartWithConfig(cfg)

	return nil
}

// StartWithKubeconfig starts the telemetry measurement with a specific kubeconfig configuration to watch all shoots
func (c *Telemetry) StartWithKubeconfig(kubeconfig clientcmd.ClientConfig, resultDir string) error {
	if _, err := os.Stat(resultDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(resultDir, os.ModePerm); err != nil {
			return err
		}
	}

	cfg := &config.Config{
		KubeConfig:     kubeconfig,
		CheckInterval:  c.interval,
		OutputDir:      resultDir,
		DisableAnalyse: true,
		ShootsFilter:   c.shootsFilter,
	}

	c.StartWithConfig(cfg)

	return nil
}

// StartForShoot starts the telemetry measurement with a kubeconfig for a specific shoot
func (c *Telemetry) StartForShoot(shootName, shootNamespace, kubeconfigPath, resultDir string) error {
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return err
	}

	c.shootsFilter = map[string]bool{
		common.GetShootKey(shootName, shootNamespace): true,
	}

	cfg := &config.Config{
		KubeConfigPath: kubeconfigPath,
		CheckInterval:  c.interval,
		OutputDir:      resultDir,
		DisableAnalyse: true,
		ShootsFilter:   c.shootsFilter,
	}

	c.StartWithConfig(cfg)

	return nil
}

// StartForShoots starts the telemetry measurement with a kubeconfig for specific shoots
func (c *Telemetry) StartForShoots(kubeconfigPath, resultDir string, shootKeys []string) error {
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		return err
	}

	c.shootsFilter = make(map[string]bool, len(shootKeys))
	for _, key := range shootKeys {
		c.shootsFilter[key] = true
	}

	cfg := &config.Config{
		KubeConfigPath: kubeconfigPath,
		CheckInterval:  c.interval,
		OutputDir:      resultDir,
		DisableAnalyse: true,
		ShootsFilter:   c.shootsFilter,
	}

	c.StartWithConfig(cfg)

	return nil
}

func (c *Telemetry) StartWithConfig(cfg *config.Config) {
	c.stopCh = make(chan struct{})
	c.signalCh = make(chan os.Signal)
	c.started = true

	go func() {
		defer close(c.stopCh)
		if err := controller.StartController(cfg, c.signalCh); err != nil {
			c.err = err
			return
		}
	}()
}

// IsStarted indicates if the controller is already running
func (c *Telemetry) IsStarted() bool {
	return c.started
}

// AddShoot adds another shoot to watch
func (c *Telemetry) AddShoot(shootKey string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	c.shootsFilter[shootKey] = true
}

// RemoveShoot removes a shoot from the telemetry watch
func (c *Telemetry) RemoveShoot(shootKey string) {
	c.mut.Lock()
	defer c.mut.Unlock()
	delete(c.shootsFilter, shootKey)
}

// HasShoot returns true if a shoot is measured
func (c *Telemetry) HasShoot(shootKey string) bool {
	c.mut.Lock()
	defer c.mut.Unlock()
	measured, ok := c.shootsFilter[shootKey]
	if !ok {
		return false
	}
	return measured
}

// WatchedShoots returns the number of monitored shoots.
func (c *Telemetry) ShootsLen() int {
	c.mut.Lock()
	defer c.mut.Unlock()
	return len(c.shootsFilter)
}

// StopAndAnalyze stops the telemetry measurement and generates a result summary
func (c *Telemetry) StopAndAnalyze(resultDir, format string) (string, map[string]*analyse.Figures, error) {
	if err := c.Stop(); err != nil {
		return "", nil, err
	}
	return c.Analyze(resultDir, format)
}

// WriteOutput forces the telemetry controller to write in memory data to file
func (c *Telemetry) WriteOutput() {
	c.signalCh <- syscall.SIGUSR1
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
	c.started = false
	return nil
}

// Analyze analyzes the previously measured values and returns the path to the summary
func (c *Telemetry) Analyze(resultDir, format string) (string, map[string]*analyse.Figures, error) {
	c.log.V(3).Info("analyze telemetry metrics")
	summaryOutput := ""
	if resultDir != "" {
		summaryOutput = path.Join(resultDir, "summary.json")
	}

	figures, err := analyse.AnalyseDir(resultDir, summaryOutput, format)
	if err != nil {
		return "", nil, errors.Wrap(err, "unable to analyze measurement")
	}
	return summaryOutput, figures, nil
}
