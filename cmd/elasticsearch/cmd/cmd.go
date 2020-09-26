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

package cmd

import (
	"fmt"
	"os"

	"github.com/gardener/test-infra/cmd/elasticsearch/cmd/precompute"
	"github.com/gardener/test-infra/pkg/logger"

	"github.com/spf13/cobra"
)

var RootFlags struct {
	Endpoint string
	User     string
	Password string
}

var rootCmd = &cobra.Command{
	Use:   "elasticsearch",
	Short: "Elasticsearch tool for TestMachinery",
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		log, err := logger.NewCliLogger()
		if err != nil {
			return err
		}
		logger.SetLogger(log)

		_ = cmd.MarkFlagRequired("endpoint")
		_ = cmd.MarkFlagRequired("user")
		_ = cmd.MarkFlagRequired("password")

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

	rootCmd.PersistentFlags().String("endpoint", "", "Elasticsearch endpoint, e.g. https://example.com:9200")
	rootCmd.PersistentFlags().StringVar(&RootFlags.User, "user", "", "Elasticsearch basic auth username")
	rootCmd.PersistentFlags().StringVar(&RootFlags.Password, "password", "", "Elasticsearch basic auth password")

	precompute.AddCommand(rootCmd)
}
