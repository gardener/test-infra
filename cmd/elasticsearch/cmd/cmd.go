// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gardener/test-infra/cmd/elasticsearch/cmd/ingest"
	"github.com/gardener/test-infra/cmd/elasticsearch/cmd/precompute"
	"github.com/gardener/test-infra/pkg/logger"
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
	ingest.AddCommand(rootCmd)
}
