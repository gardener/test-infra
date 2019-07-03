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
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/elasticsearch"
	"github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Output takes a completed testrun status and writes the results to elastic search bulk json file.
func Output(config *Config, tmClient kubernetes.Interface, namespace string, tr *tmv1beta1.Testrun, metadata *testrunner.Metadata) error {

	metadata.Testrun.StartTime = tr.Status.StartTime

	trSummary, summaries, err := DetermineTestrunSummary(tr, metadata, config)
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
			log.Warnf("Cannot get exported Test results of steps: %s", err.Error())
		} else {
			exportedDocumentsBulk := getExportedDocuments(osConfig, tr.Status, metadata)
			summaryBulk = append(summaryBulk, exportedDocumentsBulk...)
		}
	}

	summary, err := summaryBulk.Marshal()
	if err != nil {
		return err
	}

	// Print out the summary if no outputfile is specified
	if config.OutputDir == "" {
		log.Infof("Collected summary:\n%s", summary)
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

	log.Infof("Successfully written output to dir %s", outputDirPath)
	return nil
}

// DetermineTestrunSummary parses a testruns status and returns
func DetermineTestrunSummary(tr *tmv1beta1.Testrun, metadata *testrunner.Metadata, config *Config) (testrunner.TestrunSummary, []testrunner.StepSummary, error) {
	status := tr.Status
	testsRun := 0
	summaries := make([]testrunner.StepSummary, 0)

	for _, step := range status.Steps {

		stepMetadata := *metadata
		stepMetadata.Configuration = make(map[string]string, 0)
		for _, elem := range step.TestDefinition.Config {
			if elem.Type == tmv1beta1.ConfigTypeEnv && elem.Value != "" {
				stepMetadata.Configuration[elem.Name] = elem.Value
			}
		}

		summary := testrunner.StepSummary{
			Metadata:    &stepMetadata,
			Type:        testrunner.SummaryTypeTeststep,
			Name:        step.TestDefinition.Name,
			StepName:    step.Name,
			Annotations: step.Annotations,
			Phase:       step.Phase,
			StartTime:   step.StartTime,
			Duration:    step.Duration,
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

func getExportedDocuments(cfg *testmachinery.ObjectStoreConfig, status tmv1beta1.TestrunStatus, metadata *testrunner.Metadata) elasticsearch.BulkList {

	minioClient, err := minio.New(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.SSL)
	if err != nil {
		log.Errorf("Error creating minio client %s: %s", cfg.Endpoint, err.Error())
		return nil
	}

	ok, err := minioClient.BucketExists(cfg.BucketName)
	if err != nil {
		log.Errorf("Error getting bucket name %s: %s", cfg.BucketName, err.Error())
		return nil
	}
	if !ok {
		log.Errorf("Bucket %s does not exist", cfg.BucketName)
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
				log.Warnf("cannot get exportet artifact %s: %s", step.ExportArtifactKey, err.Error())
				continue
			}
			defer func() {
				err := reader.Close()
				if err != nil {
					log.Warn(err)
				}
			}()

			info, err := reader.Stat()
			if err != nil {
				log.Warnf("cannot get exported artifact %s: %s", step.ExportArtifactKey, err.Error())
				continue
			}

			if info.Size > 20 {

				files, err := getFilesFromTar(reader)
				if err != nil {
					log.Warnf("cannot untar artifact %s: %s", step.ExportArtifactKey, err.Error())
					continue
				}

				for _, doc := range files {
					bulks = append(bulks, elasticsearch.ParseExportedFiles(strings.ToLower(step.TestDefinition.Name), stepMeta, doc)...)
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
		return nil, fmt.Errorf("Cannot create gzip reader %s", err.Error())
	}

	tarReader := tar.NewReader(gzr)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Cannot read tar %s", err.Error())
		}

		if header.Typeflag == tar.TypeReg && header.Size > 0 {
			file, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("Cannot read from file %s in tar %s", header.Name, err.Error())
			}
			files = append(files, file)
		}

	}

	return files, nil
}
