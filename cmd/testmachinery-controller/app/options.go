// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	goflag "flag"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery"
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

func (o *options) GetManagerOptions() manager.Options {
	c := o.configwatcher.GetConfiguration()

	webhookOpts := webhook.Options{
		CertDir: c.Controller.WebhookConfig.CertDir,
	}
	if !c.TestMachinery.Local {
		webhookOpts.Port = c.Controller.WebhookConfig.Port
	}

	mgrOpts := manager.Options{
		LeaderElection: c.Controller.EnableLeaderElection,
		WebhookServer:  webhook.NewServer(webhookOpts),
		Metrics: server.Options{
			BindAddress: "0",
		}, // disable the metrics serving by default
	}

	if len(c.Controller.HealthAddr) != 0 {
		mgrOpts.HealthProbeBindAddress = c.Controller.HealthAddr
	}
	if len(c.Controller.MetricsAddr) != 0 {
		mgrOpts.Metrics.BindAddress = c.Controller.MetricsAddr
	}

	return mgrOpts
}
