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
	goflag "flag"
	"fmt"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery"
	vh "github.com/gardener/test-infra/pkg/util/cmdutil/viper"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type options struct {
	log                  logr.Logger
	MetricsAddr          string
	HealthProbeAddr      string
	EnableLeaderElection bool
	MaxConcurrentSyncs   int
	WebhookServerPort    int
	WebhookCertDir       string
}

func NewOptions() *options {
	return &options{}
}

func (o *options) AddFlags(fs *flag.FlagSet) {
	viperHelper := vh.NewViperHelper(nil, "config", "$HOME/.tm-bot", ".")
	vh.SetViper(viperHelper)
	viperHelper.InitFlags(fs)

	fs.StringVar(&o.MetricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	fs.StringVar(&o.HealthProbeAddr, "health-addr", ":8081", "The address the metric endpoint binds to.")
	fs.BoolVar(&o.EnableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	fs.IntVar(&o.MaxConcurrentSyncs, "max-concurrent-syncs", 1, "Max number of concurrent reconciliations.")
	fs.IntVar(&o.WebhookServerPort, "webhook-port", 443, "Specify the port where the webhook should be created")
	fs.StringVar(&o.WebhookCertDir, "webhook-cert-dir", "", "The directory that contains the webhook server key and certificate.")
	logger.InitFlags(fs)
	testmachinery.InitFlags(fs)

	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	viperHelper.BindPFlags(fs, "")
}

// Complete parses all options and flags and initializes the basic functions
func (o *options) Complete() error {
	if err := vh.ViperHelper.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			break
		default:
			return err
		}
	}

	log, err := logger.New(nil)
	if err != nil {
		return err
	}
	o.log = log.WithName("setup")
	logger.SetLogger(log)
	ctrl.SetLogger(log)

	if err = testmachinery.Setup(); err != nil {
		return errors.Wrap(err, "unable to setup testmachinery")
	}

	fmt.Println(testmachinery.GetConfig().String())
	return nil
}

func (o *options) GetManagerOptions() manager.Options {
	opts := ctrl.Options{
		LeaderElection: o.EnableLeaderElection,
		CertDir:        o.WebhookCertDir,
	}

	if !testmachinery.GetConfig().Local {
		opts.HealthProbeBindAddress = o.HealthProbeAddr
		opts.MetricsBindAddress = o.MetricsAddr
	}

	return opts
}
