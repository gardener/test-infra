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

package result

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"github.com/Masterminds/semver"
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/elasticsearch"
	"github.com/go-logr/logr"
	"github.com/minio/minio-go"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Output takes a completed testrun status and writes the results to elastic search bulk json file.
func Output(log logr.Logger, config *Config, tmClient kubernetes.Interface, namespace string, tr *tmv1beta1.Testrun, metadata *testrunner.Metadata) error {

	metadata.Testrun.StartTime = tr.Status.StartTime
	metadata.Annotations = tr.Annotations

	trSummary, summaries, err := DetermineTestrunSummary(tr, metadata, config, tmClient, log)
	if err != nil {
		return err
	}
	trStatusSummaries, err := marshalAndAppendSummaries(trSummary, summaries)
	if err != nil {
		return err
	}

	summaryMetadata := elasticsearch.ESMetadata{
		Index: elasticsearch.ESIndex{
			Index: "testmachinery",
			Type:  "_doc",
		},
	}
	summaryBulk := elasticsearch.NewList(summaryMetadata, trStatusSummaries)

	if config.S3Endpoint != "" {
		osConfig, err := getOSConfig(tmClient, namespace, config.S3Endpoint, config.S3SSL)
		if err != nil {
			log.Error(err, "cannot get exported test results of steps")
		} else {
			exportedDocumentsBulk := getExportedDocuments(log, osConfig, tr.Status, metadata)
			summaryBulk = append(summaryBulk, exportedDocumentsBulk...)
		}
	}

	summary, err := summaryBulk.Marshal()
	if err != nil {
		return err
	}

	// Print out the summary if no outputfile is specified
	if config.OutputDir == "" {
		fmt.Printf("Collected summary:\n%s", summary)
		return nil
	}

	outputDirPath, err := filepath.Abs(config.OutputDir)
	if err != nil {
		return err
	}
	err = writeBulks(outputDirPath, summary)
	if err != nil {
		return err
	}

	log.Info("successfully written output", "dir", outputDirPath)
	return nil
}

// DetermineTestrunSummary parses a testruns status and returns
func DetermineTestrunSummary(tr *tmv1beta1.Testrun, metadata *testrunner.Metadata, config *Config, tmClient kubernetes.Interface, log logr.Logger) (testrunner.TestrunSummary, []testrunner.StepSummary, error) {
	status := tr.Status
	testsRun := 0
	summaries := make([]testrunner.StepSummary, 0)

	for _, step := range status.Steps {

		stepMetadata := *metadata
		stepMetadata.Configuration = make(map[string]string, 0)
		stepMetadata.Annotations = utils.MergeStringMaps(stepMetadata.Annotations, step.Annotations)
		for _, elem := range step.TestDefinition.Config {
			if elem.Type == tmv1beta1.ConfigTypeEnv && elem.Value != "" {
				stepMetadata.Configuration[elem.Name] = elem.Value
			}
		}
		pre := preComputeTeststepFields(step, stepMetadata, tmClient, log)

		summary := testrunner.StepSummary{
			Metadata:    &stepMetadata,
			Type:        testrunner.SummaryTypeTeststep,
			Name:        step.TestDefinition.Name,
			StepName:    step.Position.Step,
			Phase:       step.Phase,
			StartTime:   step.StartTime,
			Duration:    step.Duration,
			PreComputed: pre,
		}

		summaries = append(summaries, summary)
		if step.Phase != argov1.NodeSkipped {
			testsRun++
		}
	}

	trSummary := testrunner.TestrunSummary{
		Metadata:  metadata,
		Type:      testrunner.SummaryTypeTestrun,
		Phase:     status.Phase,
		StartTime: status.StartTime,
		Duration:  status.Duration,
		TestsRun:  testsRun,
	}

	return trSummary, summaries, nil
}

func getExportedDocuments(log logr.Logger, cfg *testmachinery.S3Config, status tmv1beta1.TestrunStatus, metadata *testrunner.Metadata) elasticsearch.BulkList {

	minioClient, err := minio.New(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.SSL)
	if err != nil {
		log.Error(err, "unable to create s3 client", "endpoint", cfg.Endpoint)
		return nil
	}

	ok, err := minioClient.BucketExists(cfg.BucketName)
	if err != nil {
		log.Error(err, "error getting bucket name", "bucket", cfg.BucketName)
		return nil
	}
	if !ok {
		log.Error(fmt.Errorf("bucket %s does not exist", cfg.BucketName), "", "bucket", cfg.BucketName)
		return nil
	}

	bulks := make(elasticsearch.BulkList, 0)
	for _, step := range status.Steps {
		if step.Phase != argov1.NodeSkipped && step.ExportArtifactKey != "" {
			stepMeta := &testrunner.StepExportMetadata{
				Metadata:    *metadata,
				TestDefName: step.TestDefinition.Name,
				Phase:       step.Phase,
				StartTime:   step.StartTime,
				Duration:    step.Duration,
				PodName:     step.PodName,
			}
			reader, err := minioClient.GetObject(cfg.BucketName, step.ExportArtifactKey, minio.GetObjectOptions{})
			if err != nil {
				log.Info(fmt.Sprintf("cannot get exportet artifact %s", err.Error()), "artifact", step.ExportArtifactKey)
				continue
			}
			defer func() {
				err := reader.Close()
				if err != nil {
					log.Info("cannot close reader", "artifact", step.ExportArtifactKey)
				}
			}()

			info, err := reader.Stat()
			if err != nil {
				log.Info(fmt.Sprintf("cannot get exportet artifact %s", err.Error()), "artifact", step.ExportArtifactKey)
				continue
			}

			if info.Size > 20 {

				files, err := getFilesFromTar(reader)
				if err != nil {
					log.Info(fmt.Sprintf("cannot untar artifact: %s", err.Error()), "artifact", step.ExportArtifactKey)
					continue
				}

				for _, doc := range files {
					bulks = append(bulks, elasticsearch.ParseExportedFiles(log, strings.ToLower(step.TestDefinition.Name), stepMeta, doc)...)
				}
			}

		}
	}
	return bulks
}

func getFilesFromTar(r io.Reader) ([][]byte, error) {

	files := [][]byte{}

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("cannot create gzip reader %s", err.Error())
	}

	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cannot read tar %s", err.Error())
		}

		if header.Typeflag == tar.TypeReg && header.Size > 0 {
			file, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("cannot read from file %s in tar %s", header.Name, err.Error())
			}
			files = append(files, file)
		}

	}

	return files, nil
}

// preComputeTeststepFields precomputes fields for elasticsearch that are otherwise hard to add at runtime (i.e. as grafana does not support scripted fields)
func preComputeTeststepFields(stepStatus *tmv1beta1.StepStatus, metadata testrunner.Metadata, tmClient kubernetes.Interface, log logr.Logger) *testrunner.StepPreComputed {
	var stepEnriched testrunner.StepPreComputed

	switch stepStatus.Phase {
	case tmv1beta1.PhaseStatusFailed, tmv1beta1.PhaseStatusTimeout:
		zero := 0
		stepEnriched.PhaseNum = &zero
	case tmv1beta1.PhaseStatusSuccess:
		hundred := 100
		stepEnriched.PhaseNum = &hundred
	}

	k8sVersion := metadata.KubernetesVersion
	if k8sVersion != "" {
		semVer, err := semver.NewVersion(k8sVersion)
		if err != nil {
			log.Error(err, "cannot parse k8s Version, will not precompute derived data")
		} else {
			stepEnriched.K8SMajorMinorVersion = fmt.Sprintf("%d.%d", semVer.Major(), semVer.Minor())
		}
	}

	clusterDomain, err := testrunner.GetClusterDomainURL(tmClient)
	if err == nil {
		stepEnriched.ArgoDisplayName = "argo"
		stepEnriched.LogsDisplayName = "logs"
		stepEnriched.ClusterDomain = clusterDomain
	}

	return &stepEnriched
}
