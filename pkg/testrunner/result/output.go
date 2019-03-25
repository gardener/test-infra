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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"path"
	"strings"

	"github.com/gardener/test-infra/pkg/testrunner"

	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testrunner/elasticsearch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	minio "github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// Output takes a completed testrun status and writes the results to elastic search bulk json file.
func Output(config *Config, tmClient kubernetes.Interface, namespace string, tr *tmv1beta1.Testrun, metadata *testrunner.Metadata) error {
	if config.OutputFile == "" {
		return nil
	}

	metadata.Testrun.StartTime = tr.Status.StartTime

	if config.ArgoUIEndpoint != "" && tr.Status.Workflow != "" {
		if u, err := url.ParseRequestURI(config.ArgoUIEndpoint); err == nil {
			u.Path = path.Join(u.Path, "workflows", namespace, tr.Status.Workflow)
			metadata.ArgoUIExternalURL = u.String()
		} else {
			log.Debugf("Cannot parse Url %s: %s", config.ArgoUIEndpoint, err.Error())
		}
	}

	trSummary, err := getTestrunSummary(tr, metadata)
	if err != nil {
		return err
	}
	trSummary.Metadata = elasticsearch.ESMetadata{
		Index: elasticsearch.ESIndex{
			Index: "testmachinery",
			Type:  "_doc",
		},
	}
	SummaryBuffer, err := trSummary.Marshal()
	if err != nil {
		return err
	}

	osConfig, err := getOSConfig(tmClient, namespace, config.S3Endpoint, config.S3SSL)
	if err != nil {
		log.Warnf("Cannot get exported Test results of steps: %s", err.Error())
	} else {
		exportedDocuments := getExportedDocuments(osConfig, tr.Status, metadata)
		SummaryBuffer.Write(exportedDocuments)
	}

	err = writeToFile(config.OutputFile, SummaryBuffer.Bytes())
	if err != nil {
		return err
	}

	log.Infof("Successfully written output to file %s", config.OutputFile)
	return nil
}

func getTestrunSummary(tr *tmv1beta1.Testrun, metadata *testrunner.Metadata) (*elasticsearch.Bulk, error) {

	status := tr.Status
	testsRun := 0
	summaries := [][]byte{}

	for _, steps := range status.Steps {
		for _, step := range steps {
			summary := testrunner.StepSummary{
				Metadata:  metadata,
				Type:      testrunner.SummaryTypeTeststep,
				Name:      step.TestDefinition.Name,
				Phase:     step.Phase,
				StartTime: step.StartTime,
				Duration:  step.Duration,
			}
			encSummary, err := json.Marshal(summary)
			if err != nil {
				return nil, fmt.Errorf("Cannot marshal ElasticsearchBulk %s", err.Error())
			}
			summaries = append(summaries, encSummary)
			if step.Phase != argov1.NodeSkipped {
				testsRun++
			}
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

	encTrSummary, err := json.Marshal(trSummary)
	if err != nil {
		return nil, fmt.Errorf("Cannot marshal ElasticsearchBulk %s", err.Error())
	}

	return &elasticsearch.Bulk{
		Sources: append([][]byte{encTrSummary}, summaries...),
	}, nil

}

func getExportedDocuments(cfg *testmachinery.ObjectStoreConfig, status tmv1beta1.TestrunStatus, metadata *testrunner.Metadata) []byte {

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

	documents := bytes.NewBuffer([]byte{})
	for _, steps := range status.Steps {
		for _, step := range steps {
			if step.Phase != argov1.NodeSkipped {
				stepMeta := &testrunner.StepExportMetadata{
					Metadata:    *metadata,
					TestDefName: step.TestDefinition.Name,
					Phase:       step.Phase,
					StartTime:   step.StartTime,
					Duration:    step.Duration,
				}
				reader, err := minioClient.GetObject(cfg.BucketName, step.ExportArtifactKey, minio.GetObjectOptions{})
				if err != nil {
					log.Errorf("Cannot get exportet artifact %s: %s", step.ExportArtifactKey, err.Error())
					continue
				}
				defer reader.Close()

				info, err := reader.Stat()
				if err != nil {
					log.Errorf("Cannot get exportet artifact %s: %s", step.ExportArtifactKey, err.Error())
					continue
				}

				if info.Size > 20 {

					files, err := getFilesFromTar(reader)
					if err != nil {
						log.Errorf("Cannot untar artifact %s: %s", step.ExportArtifactKey, err.Error())
						continue
					}

					for _, doc := range files {
						documents.Write(elasticsearch.ParseExportedFiles(strings.ToLower(step.TestDefinition.Name), stepMeta, doc))
					}
				}

			}
		}
	}
	return documents.Bytes()
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

func writeToFile(fielPath string, data []byte) error {

	err := ioutil.WriteFile(fielPath, data, 0644)
	if err != nil {
		return fmt.Errorf("Cannot write to %s", err.Error())
	}

	return nil
}

func getOSConfig(tmClient kubernetes.Interface, namespace, minioEndpoint string, ssl bool) (*testmachinery.ObjectStoreConfig, error) {
	ctx := context.Background()
	defer ctx.Done()

	minioConfig := &corev1.ConfigMap{}
	err := tmClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: "tm-config"}, minioConfig)
	if err != nil {
		return nil, fmt.Errorf("Cannot get ConfigMap 'tm-config': %s", err.Error())
	}
	minioSecrets := &corev1.Secret{}
	err = tmClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: minioConfig.Data["objectstore.secretName"]}, minioSecrets)
	if err != nil {
		return nil, fmt.Errorf("Cannot get Secret '%s': %s", minioConfig.Data["objectstore.secretName"], err.Error())
	}

	return &testmachinery.ObjectStoreConfig{
		Endpoint:   minioEndpoint,
		SSL:        ssl,
		AccessKey:  string(minioSecrets.Data["accessKey"]),
		SecretKey:  string(minioSecrets.Data["secretKey"]),
		BucketName: minioConfig.Data["objectstore.bucketName"],
	}, nil
}
