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
	"os"

	goflag "flag"
	flag "github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/logger"

	"github.com/gardener/test-infra/pkg/version"

	"github.com/gardener/test-infra/pkg/controller"
	"github.com/gardener/test-infra/pkg/controller/admission/server"
	"github.com/joho/godotenv"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/gardener/test-infra/pkg/testmachinery"

	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	metricsAddr          string
	enableLeaderElection bool

	setupLogger = ctrl.Log.WithName("setup")
)

func main() {
	setupLogger.Info(fmt.Sprintf("start Test Machinery with version %s", version.Get().String()))

	if testmachinery.IsRunInsecure() {
		setupLogger.Info("testmachinery is running in insecure mode")
	}

	setupLogger.Info("setting up manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})
	if err != nil {
		setupLogger.Error(err, "unable to setup manager")
		os.Exit(1)
	}

	tmController, err := controller.New(mgr, ctrl.Log.WithName("controllers").WithName("Testrun"))
	if err != nil {
		setupLogger.Error(err, "unable to create controller", "controllers", "Testrun")
		os.Exit(1)
	}
	err = tmController.RegisterWatches()
	if err != nil {
		setupLogger.Error(err, "unable to register watches", "controllers", "Testrun")
		os.Exit(1)
	}

	if !testmachinery.GetConfig().Local {
		go server.Serve(context.Background(), ctrl.Log.WithName("admission"))
		server.UpdateHealth(true)
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
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	logger.InitFlags(nil)
	testmachinery.InitFlags(nil)
	server.InitFlags(nil)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	log, err := logger.New(nil)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}
	ctrl.SetLogger(log)

	err = godotenv.Load()
	if err == nil {
		setupLogger.Info(".env file loaded")
	} else {
		setupLogger.Info("Error loading .env file: %s", err.Error())
	}

	if err = testmachinery.Setup(); err != nil {
		setupLogger.Error(err, "unable to setup testmachinery")
		os.Exit(1)
	}

	fmt.Println(testmachinery.GetConfig().String())
}
