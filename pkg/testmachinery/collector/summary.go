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

package collector

import (
	"fmt"
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/utils"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/elasticsearch/bulk"
	"path/filepath"
)

// collectSummaryAndExports takes a completed testrun status and writes the results to elastic search bulk json files.
func (c *collector) collectSummaryAndExports(path string, tr *tmv1beta1.Testrun, meta *metadata.Metadata) error {

	meta.Testrun.StartTime = tr.Status.StartTime
	meta.Annotations = tr.Annotations

	trSummary, summaries, err := c.generateSummary(tr, meta)
	if err != nil {
		return err
	}
	trStatusSummaries, err := marshalAndAppendSummaries(trSummary, summaries)
	if err != nil {
		return err
	}

	summaryMetadata := bulk.ESMetadata{
		Index: bulk.ESIndex{
			Index: "testmachinery",
			Type:  "_doc",
		},
	}
	summaryBulk := bulk.NewList(summaryMetadata, trStatusSummaries)

	if c.s3Client != nil {
		exportedDocumentsBulk := c.getExportedDocuments(tr.Status, meta)
		summaryBulk = append(summaryBulk, exportedDocumentsBulk...)
	}

	summary, err := summaryBulk.Marshal()
	if err != nil {
		return err
	}

	// Print out the summary if no output file is specified
	if path == "" {
		fmt.Printf("Collected summary:\n%s", summary)
		return nil
	}

	outputDirPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	err = writeBulks(outputDirPath, summary)
	if err != nil {
		return err
	}

	c.log.Info("successfully written output", "dir", outputDirPath)
	return nil
}

// generateSummary parses a testruns status and returns
func (c *collector) generateSummary(tr *tmv1beta1.Testrun, meta *metadata.Metadata) (metadata.TestrunSummary, []metadata.StepSummary, error) {
	status := tr.Status
	testsRun := 0
	summaries := make([]metadata.StepSummary, 0)

	for _, step := range status.Steps {

		stepMetadata := &metadata.StepSummaryMetadata{
			Metadata:    *meta,
			StepName:    step.Name,
			TestDefName: step.TestDefinition.Name,
		}
		stepMetadata.Configuration = make(map[string]string, 0)
		stepMetadata.Annotations = utils.MergeStringMaps(stepMetadata.Annotations, step.Annotations)
		for _, elem := range step.TestDefinition.Config {
			if elem.Type == tmv1beta1.ConfigTypeEnv && elem.Value != "" {
				stepMetadata.Configuration[elem.Name] = elem.Value
			}
		}
		if meta.Testrun.ExecutionGroup != "" {
			stepMetadata.Annotations[common.LabelTestrunExecutionGroup] = meta.Testrun.ExecutionGroup
		}
		pre := c.preComputeTeststepFields(step, stepMetadata.Metadata)

		summary := metadata.StepSummary{
			Metadata:    stepMetadata,
			Type:        metadata.SummaryTypeTeststep,
			Name:        step.TestDefinition.Name,
			StepName:    step.Position.Step,
			Phase:       step.Phase,
			StartTime:   step.StartTime,
			Duration:    step.Duration,
			PreComputed: pre,
			Labels:      step.TestDefinition.Labels,
		}

		summaries = append(summaries, summary)
		if step.Phase != argov1.NodeSkipped {
			testsRun++
		}
	}

	trSummary := metadata.TestrunSummary{
		Metadata:      meta,
		Type:          metadata.SummaryTypeTestrun,
		Phase:         status.Phase,
		StartTime:     status.StartTime,
		Duration:      status.Duration,
		TestsRun:      testsRun,
		TelemetryData: meta.TelemetryData,
	}

	return trSummary, summaries, nil
}

// preComputeTeststepFields precomputes fields for elasticsearch that are otherwise hard to add at runtime (i.e. as grafana does not support scripted fields)
func (c *collector) preComputeTeststepFields(stepStatus *tmv1beta1.StepStatus, meta metadata.Metadata) *metadata.StepPreComputed {
	clusterDomain, err := util.GetClusterDomainURL(c.client)
	if err != nil {
		c.log.Error(err, "Could not obtain cluster domain URL, will not pre compute dependent fields (argo-, grafana-url)")
	}

	return PreComputeTeststepFields(stepStatus.Phase, meta, clusterDomain)
}
