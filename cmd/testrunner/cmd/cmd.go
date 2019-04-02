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
	"github.com/gardener/test-infra/cmd/testrunner/cmd/collect"
	"os"

	"github.com/gardener/test-infra/cmd/testrunner/cmd/docs"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/runtemplate"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/runtestrun"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "testrunner",
	Short: "Testrunner for Test Machinery",
}

// Execute executes the testrunner cli commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalln(err.Error())
	}
}

func init() {
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	log.SetOutput(os.Stderr)

	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Set debug mode for additional output")

	runtemplate.AddCommand(rootCmd)
	runtestrun.AddCommand(rootCmd)
	collect.AddCommand(rootCmd)
	docs.AddCommand(rootCmd)
}
