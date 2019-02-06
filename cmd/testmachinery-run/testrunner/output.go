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

package testrunner

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/gardener/test-infra/cmd/testmachinery-run/testrunner/elasticsearch"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
)

// Output takes a completed testrun status and writes the results to elastic search bulk json file.
func Output(config *TestrunConfig, tr *tmv1beta1.Testrun, metadata *Metadata) error {
	var tmKubeConfigPath = config.TmKubeconfigPath
	var s3Endpoint = config.S3Endpoint
	var concourseOnErrorDir = config.ConcourseOnErrorDir

	metadata.TestrunID = tr.Name

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

	generateNotificationConfigForAlerting(tr, concourseOnErrorDir)

	osConfig, err := getOSConfig(tmKubeConfigPath, s3Endpoint)
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

func getTestrunSummary(tr *tmv1beta1.Testrun, metadata *Metadata) (*elasticsearch.Bulk, error) {

	status := tr.Status
	testsRun := 0
	summaries := [][]byte{}

	for _, steps := range status.Steps {
		for _, step := range steps {
			summary := StepSummary{
				Metadata:  metadata,
				Type:      SummaryTypeTeststep,
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

	trSummary := TestrunSummary{
		Metadata:  metadata,
		Type:      SummaryTypeTestrun,
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

// Creates a notification config file with email recipients if any test step has failed
// The config file is then evaluated by Concourse
func generateNotificationConfigForAlerting(tr *tmv1beta1.Testrun, concourseOnErrorDir string) {
	var notifyCfgContent = createNotificationString(tr)
	notifyConfigFilePath := fmt.Sprintf("%s/notify.cfg", concourseOnErrorDir)
	writeStringToFile(notifyConfigFilePath, notifyCfgContent)
	log.Infof("Successfully created file %s", notifyConfigFilePath)
}

func createNotificationString(tr *tmv1beta1.Testrun) string {
	status := tr.Status
	hasFailingSteps := false
	emailBody := "Test Machinery steps have failed in test run '" + tr.Name + "'.\n\nFailed Steps:\n"
	notifyCfgContent :=
		"email:\n" +
			"  subject: 'Test Machinery - some steps failed in test run '" + tr.Name + "'\n" +
			"  recipients:\n"

	for _, steps := range status.Steps {
		for _, step := range steps {
			if step.Phase == argov1.NodeFailed {
				if !hasFailingSteps {
					hasFailingSteps = true
				}
				emailBody = fmt.Sprintf("%s  - %s\n", emailBody, step.TestDefinition.Name)
				for _, email := range strings.Split(step.TestDefinition.RecipientsOnFailure, ",") {
					email = strings.TrimSpace(email)
					if email != "" {
						notifyCfgContent = fmt.Sprintf("%s  - %s\n", notifyCfgContent, email)
					}
				}
			}
		}
	}
	notifyCfgContent = fmt.Sprintf("%s  mail_body: >\n%s", notifyCfgContent, emailBody)

	if hasFailingSteps {
		return notifyCfgContent
	}
	return ""
}

func writeStringToFile(filepath string, content string) {
	var err = ioutil.WriteFile(filepath, []byte(content), 0644)
	if err != nil {
		log.Errorf("Saving file  %s: failed", filepath)
	}
}

func getExportedDocuments(cfg *testmachinery.ObjectStoreConfig, status tmv1beta1.TestrunStatus, metadata *Metadata) []byte {

	minioClient, err := minio.New(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, false)
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
				stepMeta := &StepExportMetadata{
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
			file := []byte{}
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

func getOSConfig(tmKubeconfigPath, minioEndpoint string) (*testmachinery.ObjectStoreConfig, error) {
	clusterClient, err := kubernetes.NewClientFromFile(tmKubeconfigPath, nil, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("Cannot create client for %s: %s", tmKubeconfigPath, err.Error())
	}
	minioConfig, err := clusterClient.GetConfigMap(namespace, "tm-config")
	if err != nil {
		return nil, fmt.Errorf("Cannot get ConfigMap 'tm-config': %s", err.Error())
	}
	minioSecrets, err := clusterClient.GetSecret(namespace, minioConfig.Data["objectstore.secretName"])
	if err != nil {
		return nil, fmt.Errorf("Cannot get Secret '%s': %s", minioConfig.Data["objectstore.secretName"], err.Error())
	}

	return &testmachinery.ObjectStoreConfig{
		Endpoint:   minioEndpoint,
		AccessKey:  string(minioSecrets.Data["accessKey"]),
		SecretKey:  string(minioSecrets.Data["secretKey"]),
		BucketName: minioConfig.Data["objectstore.bucketName"],
	}, nil
}
