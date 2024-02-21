// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	goflag "flag"
	"os"

	"github.com/go-logr/logr"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/apis/config/install"
	"github.com/gardener/test-infra/pkg/logger"
)

type options struct {
	log        logr.Logger
	configPath string

	config     *config.BotConfiguration
	restConfig *rest.Config
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

	if err := o.readConfig(); err != nil {
		return err
	}

	o.restConfig = ctrl.GetConfigOrDie()

	return nil
}

func (o *options) readConfig() error {
	data, err := os.ReadFile(o.configPath)
	if err != nil {
		return err
	}

	scheme := runtime.NewScheme()
	install.Install(scheme)
	decoder := serializer.NewCodecFactory(scheme).UniversalDecoder()

	o.config = &config.BotConfiguration{}
	if _, _, err := decoder.Decode(data, nil, o.config); err != nil {
		return err
	}
	return nil
}
