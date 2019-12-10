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

	"github.com/gardener/test-infra/pkg/shoot-telemetry/analyse"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	cfg "github.com/gardener/test-infra/pkg/shoot-telemetry/config"
	"github.com/spf13/cobra"
)

// GetAnalyseCommand return the analyse command.
func GetAnalyseCommand() *cobra.Command {
	var (
		inputPath, outputFormat, outputPath, logLevel string
		helpText                                      = "Analyse a mesasurement output file"

		cmd = &cobra.Command{
			Use:   "analyse",
			Short: helpText,
			Long:  helpText,
			Run: func(cmd *cobra.Command, args []string) {
				log, err := logger.NewCliLogger()
				if err != nil {
					fmt.Println(err.Error())
					os.Exit(1)
				}

				// Check if only a data analysis is required.
				if inputPath != "" {
					if err := cfg.ValidateAnalyse(inputPath, outputFormat); err != nil {
						log.Error(err, "invalid flag input")
						os.Exit(1)
					}
					if _, err := analyse.Analyse(inputPath, outputPath, outputFormat); err != nil {
						log.Error(err, "Error while analysing data")
						os.Exit(1)
					}
					os.Exit(0)
				}
			},
		}
	)
	cmd.Flags().StringVar(&inputPath, "input", "", "path to measurements file")
	cmd.Flags().StringVar(&outputFormat, common.CliFlagReportFormat, common.ReportOutputFormatText, common.CliFlagHelpTextReportFormat)
	cmd.Flags().StringVar(&outputPath, common.CliFlagReportOutput, "", common.CliFlagHelpTextReportFile)
	cmd.Flags().StringVar(&logLevel, common.CliFlagLogLevel, common.DefaultLogLevel, common.CliFlagHelpLogLevel)
	logger.InitFlags(cmd.PersistentFlags())
	return cmd
}
