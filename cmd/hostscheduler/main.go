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
	"fmt"
	"github.com/gardener/test-infra/cmd/hostscheduler/completion"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/gardener/test-infra/pkg/hostscheduler/gardenerscheduler"
	"github.com/gardener/test-infra/pkg/hostscheduler/gkescheduler"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/util/cmdutil"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var (
	registration *hostscheduler.Registrations
)

var hostschedulerCmd = &cobra.Command{
	Use:     "hostscheduler",
	Aliases: []string{"hs"},
	Short:   "Manage gardener host cluster for gardener tests",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		var err error
		_, err = logger.NewCliLogger()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		cmdutil.ViperHelper.ApplyConfig()
	},
}

func init() {
	cmdutil.NewViperHelper()
	cobra.OnInitialize(initConfig)
	logger.InitFlags(hostschedulerCmd.PersistentFlags())
	completion.RegisterCmd(hostschedulerCmd)

	// register hostscheduler provider
	registration = &hostscheduler.Registrations{}
	gkescheduler.Register(registration)
	gardenerscheduler.Register(registration)

	if err := registration.Apply(hostschedulerCmd); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetConfigName("hostscheduler")
	viper.AddConfigPath("$HOME/.tm")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return
		}
		fmt.Printf("Cannot read config file from %s: %v \n", viper.ConfigFileUsed(), err)
		os.Exit(1)
	}
}

func main() {
	if err := hostschedulerCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
