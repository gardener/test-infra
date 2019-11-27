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

package testrunner

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/logger"
	"os"
	"sync"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/controller"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/test-infra/pkg/util"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// ExecuteTestruns deploys it to a testmachinery cluster and waits for the testruns results
func ExecuteTestruns(log logr.Logger, config *Config, runs RunList, testrunNamePrefix string) {
	log.V(3).Info(fmt.Sprintf("Config: %+v", util.PrettyPrintStruct(config)))

	runs.Run(log.WithValues("namespace", config.Namespace), config, testrunNamePrefix)
}

// runChart deploys the testruns in parallel into the testmachinery and watches them for their completion
func (rl RunList) Run(log logr.Logger, config *Config, testrunNamePrefix string) {
	var wg sync.WaitGroup
	for i := range rl {
		if rl[i].Error != nil {
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			for attempt := 0; attempt <= config.FlakeAttempts; attempt++ {
				rl[i].Exec(log, config, testrunNamePrefix)
				rl[i].Metadata.Retries = attempt

				if rl[i].Error == nil && rl[i].Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess {
					// testrun was successful, break retry loop
					return
				}
				if attempt < config.FlakeAttempts {
					// clean status and name of testrun if it's failed to ignore it, since a retry will be initiated
					log.Info(fmt.Sprintf("testrun failed, retry %d/%d. testrun", attempt+1, config.FlakeAttempts))
					rl[i].Testrun.Status = tmv1beta1.TestrunStatus{}
					rl[i].Testrun.ObjectMeta = metav1.ObjectMeta{GenerateName: rl[i].Testrun.GetGenerateName(), Namespace: rl[i].Testrun.GetNamespace()}
				}
			}

		}(i)
	}
	wg.Wait()
	log.Info("All testruns completed.")
}

// StartWatchController starts a new controller that watches testruns
func StartWatchController(log logr.Logger, kubeconfigPath string, stopCh chan struct{}) (watch.Watch, error) {
	cLogger, err := logger.New(&logger.Config{
		Development:       false,
		Cli:               true,
		Verbosity:         -3,
		DisableStacktrace: true,
		DisableCaller:     true,
		DisableTimestamp:  false,
	})
	if err != nil {
		return nil, err
	}
	ctrl.SetLogger(cLogger)
	tmClient, err := kubernetes.NewClientFromFile("", kubeconfigPath, kubernetes.WithClientOptions(client.Options{
		Scheme: testmachinery.TestMachineryScheme,
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to build kubernetes client from file %s", kubeconfigPath)
	}
	mgr, err := manager.New(tmClient.RESTConfig(), manager.Options{
		MetricsBindAddress: "0",
		Scheme:             testmachinery.TestMachineryScheme,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to setup manager")
	}
	_, w, err := controller.NewWatchController(mgr, log)
	if err != nil {
		return nil, errors.Wrap(err, "unable to setup controller")
	}
	go func() {
		if err := mgr.Start(stopCh); err != nil {
			log.Error(err, "error while running manager")
			os.Exit(1)
		}
	}()
	return w, nil
}
