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
	goflag "flag"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/admission/webhooks"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/configwatcher"
)

type options struct {
	log           logr.Logger
	configwatcher *configwatcher.ConfigWatcher
	configPath    string
}

func NewOptions() *options {
	return &options{}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.configPath, "config", "", "Specify the path to the configuration file")
	logger.InitFlags(fs)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Run parses all options and flags and initializes the basic functions
func (o *options) Complete() error {
	log, err := logger.New(nil)
	if err != nil {
		return err
	}
	o.log = log.WithName("setup")
	logger.SetLogger(log)
	ctrl.SetLogger(log)

	o.configwatcher, err = configwatcher.New(o.log, o.configPath)
	if err != nil {
		return err
	}
	return testmachinery.Setup(o.configwatcher.GetConfiguration())
}

func (o *options) ApplyWebhooks(ctx context.Context, mgr manager.Manager) {
	config := o.configwatcher.GetConfiguration()
	if !config.TestMachinery.Local {
		webhooks.StartHealthCheck(ctx, mgr.GetAPIReader(), config.Controller.DependencyHealthCheck.Namespace, config.Controller.DependencyHealthCheck.DeploymentName, config.Controller.DependencyHealthCheck.Interval)
		o.log.Info("Setup webhooks")
		hookServer := mgr.GetWebhookServer()
		hookServer.Register("/webhooks/validate-testrun", &webhook.Admission{Handler: webhooks.NewValidator(logger.Log.WithName("validator"))})
	}
}

func (o *options) GetManagerOptions() manager.Options {
	c := o.configwatcher.GetConfiguration()
	opts := manager.Options{
		LeaderElection:     c.Controller.EnableLeaderElection,
		CertDir:            c.Controller.WebhookConfig.CertDir,
		MetricsBindAddress: "0", // disable the metrics serving by default
	}

	if len(c.Controller.HealthAddr) != 0 {
		opts.HealthProbeBindAddress = c.Controller.HealthAddr
	}
	if len(c.Controller.MetricsAddr) != 0 {
		opts.MetricsBindAddress = c.Controller.MetricsAddr
	}

	if !c.TestMachinery.Local {
		opts.Port = c.Controller.WebhookConfig.Port
	}

	return opts
}
