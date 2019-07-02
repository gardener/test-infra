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

package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/labels"
	"os"
	"path/filepath"
	"time"

	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pollIntervalSeconds int64 = 90
	maxWaitTimeMinutes  int64 = 120
)

// ScheduleNewHostShoot selects are free hibernated shoot and wakes it up
func ScheduleNewHostShoot(ctx context.Context, k8sClient kubernetes.Interface, namespace string) (*v1beta1.Shoot, error) {
	shoot := &v1beta1.Shoot{}
	interval := time.Duration(pollIntervalSeconds) * time.Second
	timeout := time.Duration(maxWaitTimeMinutes) * time.Minute

	// try to get an available until the timeout is reached
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		var err error
		shoot, err = getAvailableHost(ctx, k8sClient, namespace)
		if err != nil {
			log.Info("No host available. Trying again...")
			log.Debug(err.Error())
			return false, nil
		}

		log.Infof("Shoot %s was selected as host and will be woken up", shoot.Name)

		if err := downloadHostKubeconfig(ctx, k8sClient, shoot); err != nil {
			log.Error(err.Error())
			if err := resetShoot(ctx, k8sClient, shoot); err != nil {
				return false, fmt.Errorf("unable to reset shoot %s: %s", shoot.Name, err.Error())
			}
			return false, nil
		}

		if err := writeHostInformationToFile(shoot); err != nil {
			log.Error(err.Error())
			if err := resetShoot(ctx, k8sClient, shoot); err != nil {
				return false, fmt.Errorf("unable to reset shoot %s: %s", shoot.Name, err.Error())
			}
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return shoot, nil
}

func getAvailableHost(ctx context.Context, k8sClient kubernetes.Interface, namespace string) (*v1beta1.Shoot, error) {
	shoots := &v1beta1.ShootList{}
	selector := labels.SelectorFromSet(labels.Set(map[string]string{ShootLabel: "true"}))
	err := k8sClient.Client().List(ctx, shoots, client.UseListOptions(&client.ListOptions{
		LabelSelector: selector,
		Namespace:     namespace,
	}))
	if err != nil {
		return nil, fmt.Errorf("shoots cannot be listed: %s", err.Error())
	}

	for _, shoot := range shoots.Items {

		// Try to use the next shoot if the current shoot is not ready.
		if shootReady(&shoot) != nil {
			log.Debugf("Shoot %s not ready. Skipping...", shoot.Name)
			continue
		}

		// if shoot is hibernated it is ready to be used as host for a test.
		// then the hibernated shoot is woken up and the gardener tests can start
		if shoot.Spec.Hibernation != nil && shoot.Spec.Hibernation.Enabled {
			shoot.Spec.Hibernation.Enabled = false

			err = k8sClient.Client().Update(ctx, &shoot)
			if err != nil {
				// shoot could not be updated, maybe it was concurrently updated by another test.
				// therefore we try the next shoot
				log.Debugf("Shoot %s cannot be updated. Skipping...", shoot.Name)
				log.Debug(err.Error())
				continue
			}
			return &shoot, nil
		}
	}
	return nil, fmt.Errorf("cannot find available shoots")
}

func downloadHostKubeconfig(ctx context.Context, k8sClient kubernetes.Interface, shoot *v1beta1.Shoot) error {
	// Write kubeconfigPath to kubeconfigPath folder: $TM_KUBECONFIG_PATH/host.config
	log.Infof("Downloading host kubeconfig to %s", HostKubeconfigPath())

	// Download kubeconfigPath secret from gardener
	secret := &corev1.Secret{}
	err := k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: ShootKubeconfigSecretName(shoot.Name)}, secret)
	if err != nil {
		return fmt.Errorf("cannot download kubeconfig for shoot %s: %s", shoot.Name, err.Error())
	}

	err = os.MkdirAll(filepath.Dir(HostKubeconfigPath()), os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot create folder %s for kubeconfig: %s", filepath.Dir(HostKubeconfigPath()), err.Error())
	}
	err = ioutil.WriteFile(HostKubeconfigPath(), secret.Data["kubeconfig"], os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot write kubeconfig to %s: %s", HostKubeconfigPath(), err.Error())
	}

	return nil
}

func writeHostInformationToFile(shoot *v1beta1.Shoot) error {
	hostConfig := client.ObjectKey{
		Name:      shoot.Name,
		Namespace: shoot.Namespace,
	}
	data, err := json.Marshal(hostConfig)
	if err != nil {
		log.Fatalf("cannot unmashal hostconfig: %s", err.Error())
	}

	err = os.MkdirAll(filepath.Dir(HostConfigFilePath()), os.ModePerm)
	if err != nil {
		log.Fatalf("cannot create folder %s for host config: %s", filepath.Dir(HostConfigFilePath()), err.Error())
	}
	err = ioutil.WriteFile(HostConfigFilePath(), data, os.ModePerm)
	if err != nil {
		log.Fatalf("cannot write host config to %s: %s", HostConfigFilePath(), err.Error())
	}

	return nil
}
