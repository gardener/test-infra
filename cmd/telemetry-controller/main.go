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
	goflag "flag"
	"fmt"
	"github.com/gardener/test-infra/pkg/logger"
	flag "github.com/spf13/pflag"
	"net/http"
	"os"

	"github.com/gardener/test-infra/pkg/version"

	"github.com/gardener/test-infra/pkg/telemetry/controller"
	"github.com/joho/godotenv"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	metricsAddr          string
	healthProbeAddr      string
	enableLeaderElection bool
	maxConcurrentSyncs   int
	cacheDir             string

	setupLogger = ctrl.Log.WithName("setup")
)

func main() {
	setupLogger.Info(fmt.Sprintf("Start Telemetry Controller with version %s", version.Get().String()))

	setupLogger.Info("setting up manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		MetricsBindAddress:     metricsAddr,
		LeaderElection:         enableLeaderElection,
		HealthProbeBindAddress: healthProbeAddr,
	})
	if err != nil {
		setupLogger.Error(err, "unable to setup manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("default", func(req *http.Request) error {
		return nil
	}); err != nil {
		setupLogger.Error(err, "unable to setup default healthz check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("default", func(req *http.Request) error {
		return nil
	}); err != nil {
		setupLogger.Error(err, "unable to setup default readyness check")
		os.Exit(1)
	}

	_, err = controller.NewTelemetryController(mgr, ctrl.Log, cacheDir, &maxConcurrentSyncs)
	if err != nil {
		setupLogger.Error(err, "unable to create controller", "controllers", "Testrun")
		os.Exit(1)
	}

	setupLogger.Info("starting the controller", "controllers", "Testrun")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		setupLogger.Error(err, "error while running manager")
		os.Exit(1)
	}

}

func init() {
	// Set commandline flags
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&healthProbeAddr, "health-addr", ":8081", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.IntVar(&maxConcurrentSyncs, "max-concurrent-syncs", 1, "Max number of concurrent reconciliations.")
	flag.StringVar(&cacheDir, "cache-dir", "/tmp/tel", "Directory to store the internal cache.")
	logger.InitFlags(nil)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	log, err := logger.New(nil)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}
	logger.SetLogger(log)
	ctrl.SetLogger(log)

	err = godotenv.Load()
	if err == nil {
		setupLogger.Info(".env file loaded")
	} else {
		setupLogger.Info("Error loading .env file: %s", err.Error())
	}
}
