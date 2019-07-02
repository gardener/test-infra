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
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HibernateShoot deletes all resources of a shoot and hibernates it afterwards
func HibernateShoot(ctx context.Context, k8sClient kubernetes.Interface) error {
	data, err := ioutil.ReadFile(HostConfigFilePath())
	if err != nil {
		return fmt.Errorf("cannot read file %s: %s", HostConfigFilePath(), err.Error())
	}

	var hostConfig client.ObjectKey
	err = json.Unmarshal(data, &hostConfig)
	if err != nil {
		return fmt.Errorf("cannot unmarshal host config: %s", err.Error())
	}

	shoot := &v1beta1.Shoot{}
	err = k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: hostConfig.Namespace, Name: hostConfig.Name}, shoot)
	if err != nil {
		return fmt.Errorf("cannot get shoot %s: %s", hostConfig.Name, err.Error())
	}

	if err := resetShoot(ctx, k8sClient, shoot); err != nil {
		log.Fatal(err.Error())
	}
	return nil
}

func resetShoot(ctx context.Context, k8sClient kubernetes.Interface, shoot *v1beta1.Shoot) error {
	log.Infof("Resetting shoot %s", shoot.Name)

	shoot, err := WaitUntilShootIsReconciled(ctx, k8sClient, shoot)
	if err != nil {
		return fmt.Errorf("cannot hibernate shoot %s: %s", shoot.Name, err.Error())
	}

	if err := cleanResources(ctx, k8sClient, shoot); err != nil {
		return err
	}

	shoot, err = WaitUntilShootIsReconciled(ctx, k8sClient, shoot)
	if err != nil {
		return fmt.Errorf("cannot hibernate shoot %s: %s", shoot.Name, err.Error())
	}

	if shoot.Spec.Hibernation == nil || shoot.Spec.Hibernation.Enabled == false {
		// Do not set any hibernation schedule as hibernation should be handled automatically by this scheduler.
		err = k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: shoot.Name}, shoot)
		if err != nil {
			return fmt.Errorf("cannot get shoot %s: %s", shoot.Name, err.Error())
		}
		shoot.Spec.Hibernation = &v1beta1.Hibernation{Enabled: true}
		err := k8sClient.Client().Update(ctx, shoot)
		if err != nil {
			return fmt.Errorf("cannot hibernate shoot %s: %s", shoot.Name, err.Error())
		}
	}
	return nil
}

func cleanResources(ctx context.Context, k8sClient kubernetes.Interface, shoot *v1beta1.Shoot) error {
	shootClient, err := kubernetes.NewClientFromSecret(k8sClient, shoot.Namespace, ShootKubeconfigSecretName(shoot.Name), client.Options{
		Scheme: kubernetes.ShootScheme,
	})
	if err != nil {
		return err
	}

	if err := CleanExtendedAPIs(ctx, shootClient.Client()); err != nil {
		return err
	}
	log.Info("Cleaned Extended API...")
	if err := CleanKubernetesResources(ctx, shootClient.Client()); err != nil {
		return err
	}
	log.Info("Cleaned Kubernetes resources...")

	return nil
}
