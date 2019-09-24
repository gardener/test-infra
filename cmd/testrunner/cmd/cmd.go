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
	"os"

	"github.com/gardener/test-infra/cmd/testrunner/cmd/collect"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/docs"
	gardener_telemetry "github.com/gardener/test-infra/cmd/testrunner/cmd/gardener_telemetry"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/run_gardener"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/run_gardener_template"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/run_template"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/run_testrun"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/version"
	"github.com/gardener/test-infra/pkg/logger"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "testrunner",
	Short: "Testrunner for Test Machinery",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		_, err = logger.NewCliLogger()
		if err != nil {
			return err
		}
		return nil
	},
}

// Execute executes the testrunner cli commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
}

func init() {

	logger.InitFlags(rootCmd.PersistentFlags())
	rootCmd.PersistentFlags().Bool("dry-run", false, "Dry run will print the rendered template")

	run_template.AddCommand(rootCmd)
	run_testrun.AddCommand(rootCmd)
	run_gardener_template.AddCommand(rootCmd)
	run_gardener.AddCommand(rootCmd)
	collectcmd.AddCommand(rootCmd)
	gardener_telemetry.AddCommand(rootCmd)
	docs.AddCommand(rootCmd)
	versioncmd.AddCommand(rootCmd)
}
