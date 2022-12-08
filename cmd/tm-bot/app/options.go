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
