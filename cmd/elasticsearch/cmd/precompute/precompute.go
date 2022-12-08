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

package precompute

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/spf13/cobra"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery/collector"
	"github.com/gardener/test-infra/pkg/util/elasticsearch"
)

var (
	// only touch ES when true, dry-run otherwise
	updateES bool
)

// AddCommand adds the precompute subcommand to another command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(precomputeCmd)
}

var precomputeCmd = &cobra.Command{
	Use:   "precompute",
	Short: "Reads existing teststep metadata, re-computes the current precomputed values and optionally updates the respective elasticsearch document.",
	PreRun: func(cmd *cobra.Command, args []string) {
		if updateES {
			logger.Log.Info("Starting 'elasticsearch precompute' in update mode", "elasticsearch endpoint", cmd.Flag("endpoint").Value, "elasticsearch user", cmd.Flag("user").Value)
		} else {
			logger.Log.Info("Starting 'elasticsearch precompute' in dry-run mode")
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
		logger.Log.Info("Finished 'elasticsearch precompute'")
	},
}

// package init defines the flags for the precompute command
func init() {
	precomputeCmd.Flags().BoolVar(&updateES, "update", false, "when false, only prints potential updates to stdout instead of touching documents in elasticsearch")
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

	pageSize := 5000
	path, payload := BuildScrollQueryInitial("testmachinery-*", pageSize)

	var processedItems int
	for {
		esResponse, err := queryAndRecomputeAndStore(esClient, path, payload)
		if err != nil {
			return err
		}
		processedItems += len(esResponse.Hits.Results)
		logger.Log.Info(fmt.Sprintf("%d%% processed (%d/%d)", processedItems*100/esResponse.Hits.Total.Value, processedItems, esResponse.Hits.Total.Value))

		// once we hit the first page with less than pageSize results, there will be no further page -> done
		if len(esResponse.Hits.Results) < pageSize {
			break
		}

		// on to the next page
		scrollID := esResponse.ScrollID
		path, payload = BuildScrollQueryNextPage(scrollID)
	}

	return nil
}

// queryAndRecomputeAndStore queries the testmachinery index for tm data, recomputes the pre-calculated fields and if changed, updates the respective data
func queryAndRecomputeAndStore(esClient elasticsearch.Client, path string, payload io.Reader) (QueryResponse, error) {
	ctx := context.Background()
	defer ctx.Done()

	bytes, err := esClient.RequestWithCtx(ctx, http.MethodGet, path, payload)
	if err != nil {
		return QueryResponse{}, err
	}

	var esResponse QueryResponse
	if err := json.Unmarshal(bytes, &esResponse); err != nil {
		return QueryResponse{}, err
	}
	logger.Log.V(6).Info("Got response from ES", "esResponse", esResponse)

	var modifiedItems []Result
	for _, result := range esResponse.Hits.Results {
		meta := result.StepSummary.Metadata
		if meta == nil {
			logger.Log.V(5).Info("Skipping entry as no metadata present, probably too old", "index", result.Index, "docID", result.DocID)
			continue
		}

		if meta.KubernetesVersion == "" && meta.Annotations != nil {
			meta.KubernetesVersion = meta.Annotations["metadata.testmachinery.gardener.cloud/k8sVersion"]
		}
		if meta.KubernetesVersion == "" && meta.Annotations != nil {
			meta.KubernetesVersion = meta.Annotations["testrunner.testmachinery.sapcloud.io/k8sVersion"]
		}

		var clusterDomain string
		oldPreComputed := result.StepSummary.PreComputed
		if oldPreComputed != nil {
			clusterDomain = oldPreComputed.ClusterDomain
		}

		newPreComputed := collector.PreComputeTeststepFields(result.StepSummary.Phase, meta.Metadata, clusterDomain)

		if reflect.DeepEqual(oldPreComputed, newPreComputed) {
			logger.Log.V(6).Info("old and new precomputed are equal -> nothing to do")
		} else {
			logger.Log.V(4).Info("preComputed field changed", "oldPreComputed", oldPreComputed, "newPreComputed", newPreComputed)
			result.StepSummary.PreComputed = newPreComputed
			modifiedItems = append(modifiedItems, result)
		}
	}

	if len(modifiedItems) == 0 {
		logger.Log.V(6).Info("no modified data for any result (in this page) -> nothing to update")
		return esResponse, nil
	}

	path, payload, err = BuildBulkUpdateQuery(modifiedItems)
	if err != nil {
		return QueryResponse{}, err
	}

	if !updateES {
		payloadBytes, err := io.ReadAll(payload)
		if err != nil {
			logger.Log.Error(err, "could not parse bulk update payload into string")
		}
		logger.Log.Info("Not modifying elasticsearch data as update flag was not set.")
		logger.Log.V(10).Info("", "path", path, "payload", string(payloadBytes))
		return esResponse, nil
	}

	// do the real update
	logger.Log.V(6).Info("going to bulk update", "path", path)
	bytes, err = esClient.RequestWithCtx(ctx, http.MethodPost, path, payload)
	if err != nil {
		logger.Log.Error(err, "Error during bulk update", "response", string(bytes))
		return QueryResponse{}, err
	}
	bulkResponse := BulkResponse{}
	if err := json.Unmarshal(bytes, &bulkResponse); err != nil {
		return QueryResponse{}, err
	}
	// check for errors in the bulk
	if bulkResponse.ErrorsOccurred {
		logger.Log.Error(errors.New("one or more errors occurred during bulk update"), "Errors occurred during bulk update")
		for _, item := range bulkResponse.Items {
			logger.Log.Info("bulk item error", "errorType", item.Index.Error.Type, "errorReason", item.Index.Error.Reason, "httpStatus", item.Index.HTTPStatus, "index", item.Index.Index, "id", item.Index.ID)
		}
		// in bulk updates a few items could go wrong, hence we'd better not stop the whole process here but continue, maybe next bulk updates will succeed ¯\_(ツ)_/¯
	}

	return esResponse, nil
}
