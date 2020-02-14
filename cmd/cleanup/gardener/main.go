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

package main

import (
	"context"
	"fmt"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/test-infra/pkg/logger"
	flag "github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gardenercommon "github.com/gardener/gardener/pkg/operation/common"
)

var (
	kubeconfigPath string

	pollInterval = 30 * time.Second
	timeout      = 30 * time.Minute
)

func init() {
	logger.InitFlags(nil)

	// configuration flags
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the gardener cluster kubeconfigPath")
}

func main() {
	flag.Parse()

	ctx := context.Background()
	defer ctx.Done()

	log, err := logger.NewCliLogger()
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}
	logger.SetLogger(log)

	// if file does not exist we exit with 0 as this means that gardener wasn't deployed
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		logger.Log.Error(nil, "gardener kubeconfig does not exists", "file", kubeconfigPath)
		os.Exit(0)
	}

	k8sClient, err := kubernetes.NewClientFromFile("", kubeconfigPath, kubernetes.WithClientOptions(client.Options{
		Scheme: kubernetes.GardenScheme,
	}))
	if err != nil {
		logger.Log.Error(err, "cannot build config from path", "file", kubeconfigPath)
		os.Exit(1)
	}

	shoots := &gardencorev1beta1.ShootList{}
	err = k8sClient.Client().List(ctx, shoots)
	if err != nil {
		logger.Log.Error(err, "cannot fetch shoots from gardener")
		os.Exit(1)
	}

	shootQueue := make(map[*gardencorev1beta1.Shoot]bool, 0)
	for _, s := range shoots.Items {
		shoot := s
		shootQueue[&shoot] = false
	}

	err = wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		for shoot, deleted := range shootQueue {
			log := logger.Log.WithValues("name", shoot.Name, "namespace", shoot.Namespace)
			if !deleted {
				log.Info("delete shoot")
				err = deleteShoot(ctx, k8sClient, shoot)
				if err != nil {
					log.Error(err, "unable to delete shoot")
					continue
				}
				shootQueue[shoot] = true
			}

			newShoot := &gardencorev1beta1.Shoot{}
			err = k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: shoot.Name}, newShoot)
			if err != nil {
				if errors.IsNotFound(err) {
					delete(shootQueue, shoot)
					continue
				}
			}

			log.Info(fmt.Sprintf("%d%%: Shoot state: %s, Description: %s; Waiting for shoot %s in namespace %s to be deleted...",
				newShoot.Status.LastOperation.Progress, newShoot.Status.LastOperation.State, newShoot.Status.LastOperation.Description, shoot.Name, shoot.Namespace))

		}
		if len(shootQueue) != 0 {
			logger.Log.Info(fmt.Sprintf("%d shoots are left to cleanup...", len(shootQueue)))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		logger.Log.Error(err, "unable to delete all shoots")
		os.Exit(1)
	}

	logger.Log.Info("Successfully deleted all shoots")
}

func deleteShoot(ctx context.Context, k8sClient kubernetes.Interface, shoot *gardencorev1beta1.Shoot) error {
	oldShoot := &gardencorev1beta1.Shoot{}
	err := k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: shoot.Name}, oldShoot)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	newShoot := oldShoot.DeepCopy()
	metav1.SetMetaDataAnnotation(&newShoot.ObjectMeta, gardenercommon.ConfirmationDeletion, "true")
	patchBytes, err := kutil.CreateTwoWayMergePatch(oldShoot, newShoot)
	if err != nil {
		return fmt.Errorf("failed to patch bytes")
	}
	if err := k8sClient.Client().Patch(ctx, oldShoot, client.ConstantPatch(types.MergePatchType, patchBytes)); err != nil {
		return err
	}

	err = k8sClient.Client().Delete(ctx, newShoot)
	if err != nil {
		return err
	}

	return nil
}
