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
	"archive/tar"
	"compress/gzip"
	"fmt"
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/util/elasticsearch/bulk"
	"github.com/minio/minio-go"
	"io"
	"io/ioutil"
	"strings"
)

func (c *collector) getExportedDocuments(status tmv1beta1.TestrunStatus, meta *metadata.Metadata) bulk.BulkList {
	bulks := make(bulk.BulkList, 0)
	for _, step := range status.Steps {
		if step.Phase != argov1.NodeSkipped && step.ExportArtifactKey != "" {
			stepMeta := &metadata.StepExportMetadata{
				Metadata:    *meta,
				StepName:    step.Name,
				TestDefName: step.TestDefinition.Name,
				Phase:       step.Phase,
				StartTime:   step.StartTime,
				Duration:    step.Duration,
				PodName:     step.PodName,
			}
			reader, err := c.s3Client.GetObject(c.s3Config.BucketName, step.ExportArtifactKey, minio.GetObjectOptions{})
			if err != nil {
				c.log.Info(fmt.Sprintf("cannot get exportet artifact %s", err.Error()), "artifact", step.ExportArtifactKey)
				continue
			}
			defer func() {
				err := reader.Close()
				if err != nil {
					c.log.Info("cannot close reader", "artifact", step.ExportArtifactKey)
				}
			}()

			info, err := reader.Stat()
			if err != nil {
				c.log.Info(fmt.Sprintf("cannot get exportet artifact %s", err.Error()), "artifact", step.ExportArtifactKey)
				continue
			}

			if info.Size > 20 {

				files, err := getFilesFromTar(reader)
				if err != nil {
					c.log.Info(fmt.Sprintf("cannot untar artifact: %s", err.Error()), "artifact", step.ExportArtifactKey)
					continue
				}

				for _, doc := range files {
					bulks = append(bulks, bulk.ParseExportedFiles(c.log, strings.ToLower(step.TestDefinition.Name), stepMeta, doc)...)
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
