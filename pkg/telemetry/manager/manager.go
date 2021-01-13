// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package manager

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/analyse"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"os"
	"path"
	"sync"
	"time"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/testrunner/telemetry"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager interface {
	IsMeasured(key string, shoot client.ObjectKey) bool
	MonitorShoots(ctx context.Context, secretRef client.ObjectKey, shoot []client.ObjectKey) (string, error)
	StopAndAnalyze(key string, shoot client.ObjectKey) (*analyse.Figures, error)
}

type manager struct {
	mut    sync.Mutex
	log    logr.Logger
	client client.Client

	dataDir string

	// controllers is a cache for all telemetry controllers that watch specific gardener clusters
	controllers map[string]*telemetry.Telemetry
}

// New returns a new telemetry controller manager
func New(log logr.Logger, k8sClient client.Client, dataDir string) Manager {
	m := &manager{
		mut:         sync.Mutex{},
		log:         log,
		client:      k8sClient,
		dataDir:     dataDir,
		controllers: make(map[string]*telemetry.Telemetry),
	}
	m.startCleanupInterval()
	return m
}

func (m *manager) StopAndAnalyze(key string, shoot client.ObjectKey) (*analyse.Figures, error) {
	tc, ok := m.getController(key)
	if !ok {
		return nil, errors.Errorf("unable to find controller for key %s", key)
	}
	shootKey := common.GetShootKey(shoot.Name, shoot.Namespace)
	resultFile := common.GetResultFile(path.Join(m.dataDir, key), shootKey)
	tc.RemoveShoot(shootKey)

	// write all measured data to file
	tc.WriteOutput()
	time.Sleep(5 * time.Second)
	figures, err := analyse.Analyse(resultFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "unable to analyze resulting file %s", resultFile)
		}
		figures = map[string]*analyse.Figures{common.GetShootKey(shoot.Name, shoot.Namespace): {}}
	}

	if err := os.Remove(resultFile); err != nil {
		log.Info("unable to cleanup file", "file", resultFile)
	}

	// stop the telemetry controller if no shoots are left to monitor
	if err := m.gcController(key); err != nil {
		return nil, err
	}
	fig := figures[common.GetShootKey(shoot.Name, shoot.Namespace)]
	fig.CalculateDownPeriodStatistics()
	fig.CalculateResponseTimeStatistics()
	return fig, nil
}

func (m *manager) MonitorShoots(ctx context.Context, secretRef client.ObjectKey, shoots []client.ObjectKey) (string, error) {
	config, err := m.getKubeconfigFromSecret(ctx, secretRef)
	if err != nil {
		return "", err
	}

	controllerKey, err := ControllerKey(config)
	if err != nil {
		return "", err
	}

	tc, ok := m.getController(controllerKey)
	if !ok {
		// create a new telemetry controller if there is no previous controller for the kubernetes cluster
		tc, err = telemetry.New(m.log, 5*time.Second)
		if err != nil {
			return "", err
		}
		m.mut.Lock()
		m.controllers[controllerKey] = tc
		m.mut.Unlock()
	}

	for _, shoot := range shoots {
		tc.AddShoot(common.GetShootKey(shoot.Name, shoot.Namespace))
	}
	if !tc.IsStarted() {
		outputDir := path.Join(m.dataDir, controllerKey)
		if err := tc.StartWithKubeconfig(config, outputDir); err != nil {
			return "", err
		}
	}
	return controllerKey, nil
}

// gcController checks if the controller is still monitoring shoots and cleans it up if not
func (m *manager) gcController(key string) error {
	tc, ok := m.getController(key)
	if !ok {
		return nil
	}
	// stop the telemetry controller if no shoots are left to monitor
	if tc.ShootsLen() == 0 {
		if err := tc.Stop(); err != nil {
			return err
		}
		// clean cache
		if err := os.RemoveAll(path.Join(m.dataDir, key)); err != nil {
			return err
		}
	}
	return nil
}

// startCleanupInterval periodically checks if all controllers are still monitoring shoots
// and cleans them up if not.
func (m *manager) startCleanupInterval() {
	ticker := time.NewTimer(30 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				for key := range m.controllers {
					if err := m.gcController(key); err != nil {
						m.log.V(3).Info("unable to garbage collect controller", "controller", key, "error", err.Error())
					}
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (m *manager) IsMeasured(key string, shoot client.ObjectKey) bool {
	tc, ok := m.controllers[key]
	if !ok {
		return false
	}

	if !tc.IsStarted() {
		return false
	}

	return tc.HasShoot(common.GetShootKey(shoot.Name, shoot.Namespace))
}

// getController is the threadsafe getter for a telemetry controller with a key.
func (m *manager) getController(key string) (*telemetry.Telemetry, bool) {
	m.mut.Lock()
	defer m.mut.Unlock()
	tc, ok := m.controllers[key]
	return tc, ok
}

// getKubeconfigFromSecret reads the kubeconfig from the specified secret ref and parses it into a k8s ClientConfig
func (m *manager) getKubeconfigFromSecret(ctx context.Context, secretRef client.ObjectKey) (clientcmd.ClientConfig, error) {
	secret := &corev1.Secret{}
	if err := m.client.Get(ctx, secretRef, secret); err != nil {
		return nil, err
	}
	kubeconfigRaw, ok := secret.Data["kubeconfig"]
	if !ok {
		return nil, errors.Errorf("kubeconfig not found in secret %s", secretRef.String())
	}

	// Load the kubeconfig.
	configObj, err := clientcmd.Load(kubeconfigRaw)
	if err != nil {
		return nil, err
	} else if configObj == nil {
		return nil, errors.New("config is nil")
	}

	// Create a kubernetes client config.
	config := clientcmd.NewDefaultClientConfig(*configObj, &clientcmd.ConfigOverrides{})
	return config, nil
}

// ControllerKey returns the key for a controller which is the hash of the config host
// so that we have one controller per cluster.
func ControllerKey(config clientcmd.ClientConfig) (string, error) {
	restConfig, err := config.ClientConfig()
	if err != nil {
		return "", err
	}
	if restConfig.Host == "" {
		return "", errors.New("unable to create controller key for config")
	}

	h := sha256.New()
	if _, err := h.Write([]byte(restConfig.Host)); err != nil {
		return "", err
	}
	key := base64.StdEncoding.EncodeToString(h.Sum(nil))
	return key, nil
}
