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
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/analyse"
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"syscall"
	"time"

	v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"

	gardeninformers "github.com/gardener/gardener/pkg/client/core/informers/externalversions/core/v1beta1"

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
func StartController(config *config.Config, signalCh chan os.Signal) error {
	var (
		stopCh = make(chan struct{})

		controller = controller{
			config:  config,
			targets: map[string]*target{},
		}
		k8sClient kubernetes.Interface
		err       error
	)

	go func() {
		for sig := range signalCh {
			if sig == syscall.SIGUSR1 {
				_ = controller.generateOutput()
			} else {
				logger.Log.Info("Received interrupt signal.")
				signal.Stop(signalCh)
				close(stopCh)
				return
			}
		}
	}()

	// Setup the necessary informer factories to initialize the required informers.
	if config.KubeConfigPath != "" {
		// Read the kubeconfig from disk.
		kubeconfigRaw, err := ioutil.ReadFile(config.KubeConfigPath)
		if err != nil {
			return err
		}

		// Load the kubeconfig.
		configObj, err := clientcmd.Load(kubeconfigRaw)
		if err != nil {
			return err
		} else if configObj == nil {
			return errors.New("config is nil")
		}

		config.KubeConfig = clientcmd.NewDefaultClientConfig(*configObj, &clientcmd.ConfigOverrides{})
	}

	restConfig, err := config.KubeConfig.ClientConfig()
	if err != nil {
		return err
	} else if restConfig == nil {
		return errors.New("clientConfig is nil")
	}
	k8sClient, err = kubernetes.NewWithConfig(kubernetes.WithRESTConfig(restConfig), kubernetes.WithClientOptions(client.Options{
		Scheme: kubernetes.GardenScheme,
	}))
	if err != nil {
		return err
	}

	k8sinformersFactory, gardenInformerFactory, err := common.SetupInformerFactory(restConfig)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(common.GetResultDir(config.OutputDir), os.ModePerm); err != nil {
		return err
	}

	// Create the informers and listers.
	controller.secrets = k8sinformersFactory.Core().V1().Secrets()
	controller.projects = gardenInformerFactory.Core().V1beta1().Projects()
	controller.shootInformer = gardenInformerFactory.Core().V1beta1().Shoots().Informer()

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

	// Start job to write the measurements constantly to disk.
	go common.Waiter(func() {
		if err := controller.generateOutput(); err != nil {
			logger.Log.Error(err, "error generating output")
		}
	}, time.Second*30, true, stopCh)

	logger.Log.Info("Start Shoot telemetry controller.")

	// add initially all shoots
	if err := controller.initTargets(k8sClient); err != nil {
		return err
	}

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

	if !config.DisableAnalyse {
		// Stop the controller. Write the in memory measurements to disk.
		if err := controller.generateOutput(); err != nil {
			return err
		}
		if _, err := analyse.AnalyseDir(config.OutputDir, controller.config.AnalyseOutput, controller.config.AnalyseFormat); err != nil {
			return err
		}
		if controller.config.AnalyseOutput != "" {
			logger.Logf(logger.Log.Info, "Write report to %s", controller.config.AnalyseOutput)
		}
	}

	return nil
}

func (c *controller) addShoot(obj interface{}) {
	var shoot = obj.(*gardencorev1beta1.Shoot)
	if shoot == nil || shoot.Status.LastOperation == nil {
		return
	}
	if c.filterShoot(shoot) {
		logger.Logf(logger.Log.V(6).Info, "%s Filter shoot: %s/%s", common.LogDebugAddPrefix, shoot.Namespace, shoot.Name)
		return
	}

	// Reject Shoots which are configured to be hibernated or Shoots which should wake up are still hibernated.
	if (shoot.Spec.Hibernation != nil && shoot.Spec.Hibernation.Enabled != nil && *shoot.Spec.Hibernation.Enabled) || shoot.Status.IsHibernated {
		logger.Logf(logger.Log.V(5).Info, "%s Reject hibernated shoot: %s/%s", common.LogDebugAddPrefix, shoot.Namespace, shoot.Name)
		return
	}

	if shoot.Status.LastOperation != nil && shoot.Status.LastOperation.Type == "Reconcile" {
		logger.Logf(logger.Log.V(5).Info, "%s Add shoot to queue: %s/%s", common.LogDebugAddPrefix, shoot.Namespace, shoot.Name)
		c.addTarget(shoot)
	}
}

func (c *controller) updateShoot(oldObj, newObj interface{}) {
	var (
		oldShoot = oldObj.(*gardencorev1beta1.Shoot)
		newShoot = newObj.(*gardencorev1beta1.Shoot)
	)
	if oldShoot == nil || newShoot == nil || oldShoot.Status.LastOperation == nil || newShoot.Status.LastOperation == nil {
		return
	}
	if c.filterShoot(newShoot) {
		logger.Logf(logger.Log.V(6).Info, "%s Filter shoot: %s/%s", common.LogDebugUpdatePrefix, newShoot.Namespace, newShoot.Name)
		return
	}

	// Remove shoots which are hibernated.
	if (newShoot.Spec.Hibernation != nil && newShoot.Spec.Hibernation.Enabled != nil && *newShoot.Spec.Hibernation.Enabled) || newShoot.Status.IsHibernated {
		logger.Logf(logger.Log.V(5).Info, "%s Ignore hibernated shoot: %s/%s", common.LogDebugUpdatePrefix, newShoot.Namespace, newShoot.Name)
		c.removeTarget(oldShoot)
		return
	}

	if oldShoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeCreate && newShoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeCreate {
		if oldShoot.Status.LastOperation.Progress != newShoot.Status.LastOperation.Progress && newShoot.Status.LastOperation.Progress == 100 {
			logger.Logf(logger.Log.V(5).Info, "%s Add shoot %s/%s to the queue", common.LogDebugUpdatePrefix, newShoot.GetNamespace(), newShoot.GetName())
			c.addTarget(newShoot)
		}
		return
	}

	// Remove shoot from queue if it move from reconcile to create/delete.
	if oldShoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeReconcile && newShoot.Status.LastOperation.Type != gardencorev1beta1.LastOperationTypeReconcile {
		logger.Logf(logger.Log.V(5).Info, "%s Remove shoot %s/%s from queue", common.LogDebugUpdatePrefix, newShoot.GetNamespace(), newShoot.GetName())
		c.removeTarget(oldShoot)
		return
	}

	if newShoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeReconcile {
		// Add Shoot again if it was hibernated before and woke up again.
		if oldShoot.Status.IsHibernated {
			logger.Logf(logger.Log.V(5).Info, "%s Add awakened shoot: %s/%s", common.LogDebugUpdatePrefix, newShoot.Namespace, newShoot.Name)
			c.addTarget(newShoot)
			return
		}

		// Add Shoot if it moves from other State(Create) into Reconcile state.
		if oldShoot.Status.LastOperation.Type != gardencorev1beta1.LastOperationTypeReconcile {
			logger.Logf(logger.Log.V(5).Info, "%s Add shoot %s/%s to the queue", common.LogDebugUpdatePrefix, newShoot.GetNamespace(), newShoot.GetName())
			c.addTarget(newShoot)
			return
		}
	}
}
