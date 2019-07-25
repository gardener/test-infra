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
	"flag"
	"os"
	"time"

	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	kubeconfigPath string
	debug          bool

	pollInterval = 30 * time.Second
	timeout      = 30 * time.Minute
)

func init() {
	log.SetFormatter(&log.TextFormatter{})
	// configuration flags
	flag.StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the gardener cluster kubeconfigPath")
	flag.BoolVar(&debug, "debug", false, "debug output.")
}

func main() {
	flag.Parse()

	ctx := context.Background()
	defer ctx.Done()
	if debug {
		log.SetLevel(log.DebugLevel)
		log.Warn("Set debug log level")
	}

	// if file does not exist we exit with 0 as this means that gardener wasn't deployed
	if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
		log.Infof("gardener kubeconfig at %s does not exists", kubeconfigPath)
		os.Exit(0)
	}

	k8sClient, err := kubernetes.NewClientFromFile("", kubeconfigPath, client.Options{
		Scheme: kubernetes.GardenScheme,
	})
	if err != nil {
		log.Fatalf("cannot build config from path %s: %s", kubeconfigPath, err.Error())
	}

	shoots := &v1beta1.ShootList{}
	err = k8sClient.Client().List(ctx, shoots)
	if err != nil {
		log.Fatalf("cannot fetch shoots from gardener: %s", err.Error())
	}

	shootQueue := make(map[*v1beta1.Shoot]bool, 0)
	for _, s := range shoots.Items {
		shoot := s
		shootQueue[&shoot] = false
	}

	err = wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		for shoot, deleted := range shootQueue {
			if !deleted {
				log.Infof("Delete shoot %s in namespace %s", shoot.Name, shoot.Namespace)
				err = deleteShoot(ctx, k8sClient, shoot)
				if err != nil {
					log.Infof("unable to delete shoot %s in namespace %s: %s", shoot.Name, shoot.Namespace, err.Error())
					continue
				}
				shootQueue[shoot] = true
			}

			newShoot := &v1beta1.Shoot{}
			err = k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: shoot.Name}, newShoot)
			if err != nil {
				if errors.IsNotFound(err) {
					delete(shootQueue, shoot)
					continue
				}
			}

			log.Infof("%d%%: Shoot state: %s, Description: %s; Waiting for shoot %s in namespace %s to be deleted...",
				newShoot.Status.LastOperation.Progress, newShoot.Status.LastOperation.State, newShoot.Status.LastOperation.Description, shoot.Name, shoot.Namespace)

		}
		if len(shootQueue) != 0 {
			log.Infof("%d shoots are left to cleanup...", len(shootQueue))
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Info("Successfully deleted all shoots")
}

func deleteShoot(ctx context.Context, k8sClient kubernetes.Interface, shoot *v1beta1.Shoot) error {
	newShoot := &v1beta1.Shoot{}
	err := k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: shoot.Name}, newShoot)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// todo: replace with gardener common.ConfirmationDeletion
	newShoot.Annotations["confirmation.garden.sapcloud.io/deletion"] = "true"
	err = k8sClient.Client().Update(ctx, newShoot)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	err = k8sClient.Client().Delete(ctx, newShoot)
	if err != nil {
		return err
	}

	return nil
}
