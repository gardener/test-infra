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
	"errors"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/analyse"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	gardeninformers "github.com/gardener/gardener/pkg/client/garden/informers/externalversions/garden/v1beta1"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/config"
)

type controller struct {
	config        *config.Config
	secrets       v1.SecretInformer
	projects      gardeninformers.ProjectInformer
	shootInformer cache.SharedIndexInformer
	domain        string

	targetsMutex sync.Mutex
	targets      map[string]*target
}

// StartController initialize the telemetry controller.
func StartController(config *config.Config) error {
	var (
		stopCh   = make(chan struct{})
		signalCh = make(chan os.Signal, 2)

		controller = controller{
			config:  config,
			targets: map[string]*target{},
		}
	)

	// React on OS signals and init the shut down steps.
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-signalCh
		log.Info("Received interrupt signal.")
		signal.Stop(signalCh)
		close(stopCh)
	}()

	// Setup the necessary informer factories to initialize the required informers.
	k8sinformersFactory, gardenInformerFactory, err := common.SetupInformerFactory(config.KubeConfigPath)
	if err != nil {
		return err
	}

	// Create the informers and listers.
	controller.secrets = k8sinformersFactory.Core().V1().Secrets()
	controller.projects = gardenInformerFactory.Garden().V1beta1().Projects()
	controller.shootInformer = gardenInformerFactory.Garden().V1beta1().Shoots().Informer()

	secretInformer := controller.secrets.Informer()
	projectInformer := controller.projects.Informer()

	// Start the informer factories and wait until the informer caches has been synced.
	k8sinformersFactory.Start(stopCh)
	gardenInformerFactory.Start(stopCh)

	if !cache.WaitForCacheSync(stopCh, controller.shootInformer.HasSynced, projectInformer.HasSynced, secretInformer.HasSynced) {
		return errors.New("Failed to sync informers")
	}

	// Fetch the internal domain.
	if err := controller.fetchInternalDomain(); err != nil {
		return err
	}

	// Start job to write the measurments constantly to disk.
	go common.Waiter(func() {
		if err := controller.generateOutput(); err != nil {
			log.Error(err.Error())
		}
	}, time.Second*30, true, stopCh)

	log.Info("Start Shoot telemetry controller.")

	// Register event handlers for new Shoots and Shoot updates.
	controller.shootInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    controller.addShoot,
		UpdateFunc: controller.updateShoot,
	})

	// Start the observation of the Shoot apiserver and etcd availability.
	go common.Waiter(func() {
		controller.targetsMutex.Lock()
		for _, t := range controller.targets {
			if !t.archived && !t.active {
				go controller.observeTarget(t, stopCh)
			}
		}
		controller.targetsMutex.Unlock()
	}, time.Second*3, true, stopCh)

	<-stopCh
	// Stop the controller. Write the in memory measurements to disk.
	if err := controller.generateOutput(); err != nil {
		return err
	}

	if !config.DisableAnalyse {
		if err := analyse.Analyse(controller.config.OutputFile, controller.config.AnalyseOutput, controller.config.AnalyseFormat); err != nil {
			return err
		}
		if controller.config.AnalyseOutput != "" {
			log.Infof("Write report to %s", controller.config.AnalyseOutput)
		}
	}

	return nil
}

func (c *controller) addShoot(obj interface{}) {
	var shoot = obj.(*gardenv1beta1.Shoot)
	if shoot == nil || shoot.Status.LastOperation == nil {
		return
	}

	// Reject Shoots which are configured to be hibernated or Shoots which should wake up are still hibernated.
	if (shoot.Spec.Hibernation != nil && shoot.Spec.Hibernation.Enabled != nil && *shoot.Spec.Hibernation.Enabled) || (shoot.Status.IsHibernated != nil && *shoot.Status.IsHibernated) {
		log.Debugf("%s Reject hibernated shoot: %s/%s", common.LogDebugAddPrefix, shoot.Namespace, shoot.Name)
		return
	}

	if shoot != nil && shoot.Status.LastOperation != nil && shoot.Status.LastOperation.Type == "Reconcile" {
		log.Debugf("%s Add shoot to queue: %s/%s", common.LogDebugAddPrefix, shoot.Namespace, shoot.Name)
		c.addTarget(shoot)
	}
}

func (c *controller) updateShoot(oldObj, newObj interface{}) {
	var (
		oldShoot = oldObj.(*gardenv1beta1.Shoot)
		newShoot = newObj.(*gardenv1beta1.Shoot)
	)
	if oldShoot == nil || newShoot == nil || oldShoot.Status.LastOperation == nil || newShoot.Status.LastOperation == nil {
		return
	}

	// Remove shoots which are hibernated.
	if (newShoot.Spec.Hibernation != nil && newShoot.Spec.Hibernation.Enabled != nil && *newShoot.Spec.Hibernation.Enabled) || (newShoot.Status.IsHibernated != nil && *newShoot.Status.IsHibernated) {
		log.Debugf("%s Ignore hibernated shoot: %s/%s", common.LogDebugUpdatePrefix, newShoot.Namespace, newShoot.Name)
		c.removeTarget(oldShoot)
		return
	}

	// Remove shoot from queue if it move from reconcile to create/delete.
	if oldShoot.Status.LastOperation.Type == "Reconcile" && newShoot.Status.LastOperation.Type != "Reconcile" {
		log.Debugf("%s Remove shoot %s/%s from queue", common.LogDebugUpdatePrefix, newShoot.GetNamespace(), newShoot.GetName())
		c.removeTarget(oldShoot)
		return
	}

	if newShoot.Status.LastOperation.Type == "Reconcile" {
		// Add Shoot again if it was hibernated before and woke up again.
		if oldShoot.Status.IsHibernated != nil && *oldShoot.Status.IsHibernated {
			log.Debugf("%s Add awakened shoot: %s/%s", common.LogDebugUpdatePrefix, newShoot.Namespace, newShoot.Name)
			c.addTarget(newShoot)
			return
		}

		// Add Shoot if it moves from other State(Create) into Reconcile state.
		if oldShoot.Status.LastOperation.Type != "Reconcile" {
			log.Debugf("%s Add shoot %s/%s to the queue", common.LogDebugUpdatePrefix, newShoot.GetNamespace(), newShoot.GetName())
			c.addTarget(newShoot)
			return
		}
	}
}
