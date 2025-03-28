// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"github.com/spf13/cobra"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/util/elasticsearch"
)

var (
	// only touch ES when true, dry-run otherwise
	updateES bool
	filepath string
)

// AddCommand adds the ingest subcommand to another command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(ingestCmd)
}

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Verifies that ingestion of testrun metadata into elasticsearch/opensearch works.",
	PreRun: func(cmd *cobra.Command, args []string) {
		if updateES {
			logger.Log.Info("Starting 'elasticsearch ingest' in update mode", "elasticsearch endpoint", cmd.Flag("endpoint").Value, "elasticsearch user", cmd.Flag("user").Value)
		} else {
			logger.Log.Info("Starting 'elasticsearch ingest' in dry-run mode")
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := run(cmd); err != nil {
			logger.Log.Error(err, "error during execution")
			return err
		}
		return nil
	},
	PostRun: func(cmd *cobra.Command, args []string) {
		logger.Log.Info("Finished 'elasticsearch ingest'")
	},
}

// package init defines the flags for the ingest command
func init() {
	ingestCmd.Flags().StringVar(&filepath, "file", "", "path to a bulk ingestion file")
}

func run(cmd *cobra.Command) error {
	esClient, err := elasticsearch.NewClient(config.ElasticSearch{
		Endpoint: cmd.Flag("endpoint").Value.String(),
		Username: cmd.Flag("user").Value.String(),
		Password: cmd.Flag("password").Value.String(),
	})
	if err != nil {
		return err
	}
	err = esClient.BulkFromFile(cmd.Flag("file").Value.String())
	if err != nil {
		return err
	}

	return nil
}
