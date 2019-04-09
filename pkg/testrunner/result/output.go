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
	"io"
	"io/ioutil"
	"net/url"
	"path"
	"strings"
	"text/template"

	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/elasticsearch"
	"github.com/gardener/test-infra/pkg/util"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/minio/minio-go"
	log "github.com/sirupsen/logrus"
)

// Output takes a completed testrun status and writes the results to elastic search bulk json file.
func Output(config *Config, tmClient kubernetes.Interface, namespace string, tr *tmv1beta1.Testrun, metadata *testrunner.Metadata) error {

	metadata.Testrun.StartTime = tr.Status.StartTime

	if config.ArgoUIEndpoint != "" && tr.Status.Workflow != "" {
		if u, err := url.ParseRequestURI(config.ArgoUIEndpoint); err == nil {
			u.Path = path.Join(u.Path, "workflows", namespace, tr.Status.Workflow)
			metadata.ArgoUIExternalURL = u.String()
		} else {
			log.Debugf("Cannot parse argo Url %s: %s", config.ArgoUIEndpoint, err.Error())
		}
	}

	if config.KibanaEndpoint != "" && tr.Status.Workflow != "" {
		if u, err := url.ParseRequestURI(config.KibanaEndpoint); err == nil {
			metadata.KibanaExternalURL = buildKibanaWorkflowURL(u.String(), namespace, tr.Status.Workflow)
		} else {
			log.Debugf("Cannot parse kibana Url %s: %s", config.KibanaEndpoint, err.Error())
		}
	}

	trSummary, err := getTestrunSummary(tr, metadata)
	if err != nil {
		return err
	}

	summaryMetadata := elasticsearch.ESMetadata{
		Index: elasticsearch.ESIndex{
			Index: "testmachinery",
			Type:  "_doc",
		},
	}
	summaryBulk := elasticsearch.NewList(summaryMetadata, trSummary)

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
		// TODO: change to more readable output
		log.Info(summary)
		return nil
	}
	err = writeBulks(config.OutputDir, summary)
	if err != nil {
		return err
	}

	log.Infof("Successfully written output to file %s", config.OutputDir)
	return nil
}

func getTestrunSummary(tr *tmv1beta1.Testrun, metadata *testrunner.Metadata) ([][]byte, error) {

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
			encSummary, err := util.MarshalNoHTMLEscape(summary)
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

	encTrSummary, err := util.MarshalNoHTMLEscape(trSummary)
	if err != nil {
		return nil, fmt.Errorf("Cannot marshal ElasticsearchBulk: %s", err.Error())
	}

	return append([][]byte{encTrSummary}, summaries...), nil
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
	for _, steps := range status.Steps {
		for _, step := range steps {
			if step.Phase != argov1.NodeSkipped {
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
					log.Warnf("Cannot get exportet artifact %s: %s", step.ExportArtifactKey, err.Error())
					continue
				}
				defer reader.Close()

				info, err := reader.Stat()
				if err != nil {
					log.Warnf("Cannot get exportet artifact %s: %s", step.ExportArtifactKey, err.Error())
					continue
				}

				if info.Size > 20 {

					files, err := getFilesFromTar(reader)
					if err != nil {
						log.Warnf("Cannot untar artifact %s: %s", step.ExportArtifactKey, err.Error())
						continue
					}

					for _, doc := range files {
						bulks = append(bulks, elasticsearch.ParseExportedFiles(strings.ToLower(step.TestDefinition.Name), stepMeta, doc)...)
					}
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

// buildKibanaWorkflowURL construct a valid kibana url from the given endpoint and workflow
// typical endpoint: https://kibana.ingress.<my-cluster-domain>
// typical kibana url: https://kibana.ingress.<my-cluster-domain>/app/kibana#/...
func buildKibanaWorkflowURL(endpointPath string, namespace string, workflow string) string {
	const indexPatternID = "d1134060-5189-11e9-b7dc-3fcc67c9fd25"
	const pathTemplate = "/app/kibana#/discover?_g=(refreshInterval:(pause:!t,value:0),time:(from:now-7d,mode:quick,to:now))&_a=(columns:!(namespace,pod,container,stream,log,severity,annotations.workflows_argoproj_io%2Fnode-name),filters:!(('$state':(store:appState),meta:(alias:!n,disabled:!f,index:{{.IndexPatternID}},key:container,negate:!f,params:(query:main,type:phrase),type:phrase,value:main),query:(match:(container:(query:main,type:phrase)))),('$state':(store:appState),meta:(alias:!n,disabled:!f,index:{{.IndexPatternID}},key:labels.workflows_argoproj_io%2Fworkflow.keyword,negate:!f,params:(query:{{.WorkflowID}},type:phrase),type:phrase,value:{{.WorkflowID}}),query:(match:(labels.workflows_argoproj_io%2Fworkflow.keyword:(query:{{.WorkflowID}},type:phrase))))),index:{{.IndexPatternID}},interval:auto,query:(language:lucene,query:''),sort:!('@flb_time',asc))"
	filters := kibanaFilter{indexPatternID, workflow, ""}
	tmpl, err := template.New("kibanaLink").Parse(pathTemplate)
	if err != nil {
		log.Errorf("cannot create kibana template url from template '%s': %s", pathTemplate, err.Error())
	}
	builder := strings.Builder{}
	builder.WriteString(endpointPath)
	err = tmpl.Execute(&builder, filters)
	if err != nil {
		log.Errorf("cannot template kibana url with ns '%s' and workflow '%s': %s", namespace, workflow, err.Error())
	}
	return builder.String()
}
