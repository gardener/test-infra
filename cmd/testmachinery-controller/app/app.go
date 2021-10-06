// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/controller"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/health"
	"github.com/gardener/test-infra/pkg/version"
)

func NewTestMachineryControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "testmachinery-controller",
		Short: "TestMachinery controller manages the orchestration of test in multiple testruns",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			options.run(ctx)
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) {
	o.log.Info(fmt.Sprintf("start Test Machinery with version %s", version.Get().String()))
	fmt.Println(testmachinery.GetConfig().String())
	if testmachinery.IsRunInsecure() {
		o.log.Info("testmachinery is running in insecure mode")
	}

	o.log.Info("setting up manager")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), o.GetManagerOptions())
	if err != nil {
		o.log.Error(err, "unable to setup manager")
		os.Exit(1)
	}
	mgr.GetClient()

	if err := controller.RegisterTestMachineryController(mgr, ctrl.Log, o.configwatcher.GetConfiguration()); err != nil {
		o.log.Error(err, "unable to create controller", "controllers", "Testrun")
		os.Exit(1)
	}

	if len(o.configwatcher.GetConfiguration().Controller.HealthAddr) != 0 {
		if err := mgr.AddHealthzCheck("default", health.Healthz()); err != nil {
			o.log.Error(err, "unable to register default health check")
			os.Exit(1)
		}
	}

	o.ApplyWebhooks(ctx, mgr)

	o.log.Info("starting the controller", "controllers", "Testrun")
	if err := mgr.Start(ctx); err != nil {
		o.log.Error(err, "error while running manager")
		os.Exit(1)
	}
}
