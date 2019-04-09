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
	"github.com/gardener/test-infra/pkg/version"
	"os"

	"github.com/gardener/test-infra/pkg/controller"
	"github.com/gardener/test-infra/pkg/server"
	"github.com/joho/godotenv"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/gardener/test-infra/pkg/testmachinery"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	masterURL  string
	kubeconfig string
	local      bool
)

func main() {

	log.Infof("Start Test Machinery with Version %s", version.Get().String())
	flag.Parse()

	if testmachinery.IsRunInsecure() {
		log.Warn("testmachinery is running in insecure mode")
	}

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("cannot build config from %s", kubeconfig)
	}

	log.Info("setting up manager")
	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		log.Fatalf("Unable to set up overall controller manager: %s", err.Error())
	}

	tmController, err := controller.New(mgr)
	if err != nil {
		log.Fatalf("Cannot register controller: %s", err.Error())
	}
	err = tmController.RegisterWatches()
	if err != nil {
		log.Fatalf("Cannot register watches: %s", err.Error())
	}

	if !local {
		go server.Serve(context.Background(), mgr)
		server.UpdateHealth(true)
	}

	log.Info("Starting the Controller.")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatalf("unable to run the manager: %s", err.Error())
	}

}

func init() {
	err := godotenv.Load()
	if err == nil {
		log.Info(".env file loaded")
	} else {
		log.Warnf("Error loading .env file: %s", err.Error())
	}
	testmachinery.Setup()

	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	log.SetOutput(os.Stderr)

	if os.Getenv("LOG_LEVEL") == "debug" {
		log.SetLevel(log.DebugLevel)
		log.Warn("Set debug log level")
	}

	// Set commandline flags
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&testmachinery.GetConfig().Insecure, "insecure", false, "The test machinery runs in insecure mode which menas that local testdefs are allowed and therefore hostPaths are mounted.")
	flag.BoolVar(&local, "local", false, "The controller runs outside of a cluster.")

}
