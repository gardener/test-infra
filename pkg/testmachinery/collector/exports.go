// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/util/elasticsearch/bulk"
)

func (c *collector) getExportedDocuments(status tmv1beta1.TestrunStatus, meta *metadata.Metadata) bulk.BulkList {
	bulks := make(bulk.BulkList, 0)
	for _, step := range status.Steps {
		if step.Phase != argov1.NodeSkipped && step.ExportArtifactKey != "" {
			stepMeta := &metadata.StepExportMetadata{
				StepSummaryMetadata: metadata.StepSummaryMetadata{
					Metadata:    *meta,
					StepName:    step.Name,
					TestDefName: step.TestDefinition.Name,
				},
				Phase:     step.Phase,
				StartTime: step.StartTime,
				Duration:  step.Duration,
				PodName:   step.PodName,
			}
			reader, err := c.s3Client.GetObject(c.s3Config.BucketName, step.ExportArtifactKey)
			if err != nil {
				c.log.Info(fmt.Sprintf("cannot get exportet artifact %s", err.Error()), "artifact", step.ExportArtifactKey)
				if reader != nil {
					if err := reader.Close(); err != nil {
						c.log.Info(fmt.Sprintf("cannot close reader after artifact error: %v", err), "artifact", step.ExportArtifactKey)
					}
				}
				continue
			}

			info, err := reader.Stat()
			if err != nil {
				c.log.Info(fmt.Sprintf("cannot get exported artifact %s", err.Error()), "artifact", step.ExportArtifactKey)
				if err := reader.Close(); err != nil {
					c.log.Info(fmt.Sprintf("cannot close reader after exported artifact error: %v", err), "artifact", step.ExportArtifactKey)
				}
				continue
			}

			if info.Size > 20 {

				files, err := getFilesFromTar(reader)
				if err != nil {
					c.log.Info(fmt.Sprintf("cannot untar artifact: %s", err.Error()), "artifact", step.ExportArtifactKey)
					if err := reader.Close(); err != nil {
						c.log.Info(fmt.Sprintf("cannot close reader after untarring artifact error: %v", err), "artifact", step.ExportArtifactKey)
					}
					continue
				}

				for _, doc := range files {
					bulks = append(bulks, bulk.ParseExportedFiles(c.log, strings.ToLower(step.TestDefinition.Name), stepMeta, doc)...)
				}
			}
			if err := reader.Close(); err != nil {
				c.log.Info(fmt.Sprintf("cannot close reader after artifact error: %v", err), "artifact", step.ExportArtifactKey)
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
			file, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, fmt.Errorf("cannot read from file %s in tar %s", header.Name, err.Error())
			}
			files = append(files, file)
		}

	}

	return files, nil
}
