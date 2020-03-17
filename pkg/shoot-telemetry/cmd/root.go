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

package cmd

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/logger"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	cfg "github.com/gardener/test-infra/pkg/shoot-telemetry/config"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/controller"
	"github.com/spf13/cobra"
)

// GetRootCommand return the root command.
func GetRootCommand() *cobra.Command {
	var (
		config = cfg.Config{}

		cmd = &cobra.Command{
			Use:  "garden-shoot-telemetry",
			Long: "A telemetry controller to get granular insights of Shoot apiserver and etcd availability",
			Run: func(cmd *cobra.Command, args []string) {
				log, err := logger.NewCliLogger()
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}
				logger.SetLogger(log)

				// Parse the check interval duration.
				duration, err := time.ParseDuration(config.CheckIntervalInput)
				if err != nil {
					log.Error(err, "unable to parse duration")
					os.Exit(1)
				}
				config.CheckInterval = duration

				config.ShootsFilter = make(map[string]bool, len(config.ShootNames))
				for _, shoot := range config.ShootNames {
					config.ShootsFilter[shoot] = true
				}

				// Validate the passed flag inputs.
				if err := config.Validate(); err != nil {
					log.Error(err, "invalid flag input")
					os.Exit(1)
				}

				// React on OS signals and init the shut down steps.
				signalCh := make(chan os.Signal, 2)
				signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

				// Start the controller.
				if err := controller.StartController(&config, signalCh); err != nil {
					log.Error(err, "error while executing controller")
					os.Exit(1)
				}
			},
		}
	)

	cmd.Flags().StringVar(&config.KubeConfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "kubeconfig to target garden cluster")
	cmd.Flags().StringVar(&config.CheckIntervalInput, "interval", "5s", "frequency to check Shoot/Seed apiserver and etcd")
	cmd.Flags().StringVar(&config.OutputDir, "output", "", "directory to store the measurement file")
	cmd.Flags().BoolVar(&config.DisableAnalyse, "disable-analyse", false, "disable the analysis of the measured values")
	cmd.Flags().StringVar(&config.AnalyseFormat, common.CliFlagReportFormat, common.ReportOutputFormatText, common.CliFlagHelpTextReportFormat)
	cmd.Flags().StringVar(&config.AnalyseOutput, common.CliFlagReportOutput, "", common.CliFlagHelpTextReportFile)
	cmd.Flags().StringArrayVar(&config.ShootNames, "shoot", []string{}, "target shoots to watch. Must be of form <shoot-namespace>/<shoot-name>")

	logger.InitFlags(cmd.PersistentFlags())
	return cmd
}
