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
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util/elasticsearch"
	"github.com/go-logr/logr"
	"github.com/minio/minio-go"
	"github.com/pkg/errors"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Interface is the testmachinery interface to collects testrun results, store them in a persistent store
// and handle the metadata.
type Interface interface {
	GetMetadata(tr *tmv1beta1.Testrun) (*metadata.Metadata, error)
	Collect(tr *tmv1beta1.Testrun, metadata *metadata.Metadata) error
}

type collector struct {
	log    logr.Logger
	client client.Client

	esConfig *elasticsearch.Config
	esClient elasticsearch.Client
	s3Config *testmachinery.S3Config
	s3Client *minio.Client
}

func New(log logr.Logger, k8sClient client.Client, esConfig *elasticsearch.Config, s3Config *testmachinery.S3Config) (Interface, error) {
	c := &collector{
		log:      log,
		client:   k8sClient,
		esConfig: esConfig,
		s3Config: s3Config,
	}

	if s3Config != nil {
		minioClient, err := minio.New(s3Config.Endpoint, s3Config.AccessKey, s3Config.SecretKey, s3Config.SSL)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create s3 client for %s", s3Config.Endpoint)
		}

		ok, err := minioClient.BucketExists(c.s3Config.BucketName)
		if err != nil {
			c.log.Error(err, "error getting bucket name", "bucket", c.s3Config.BucketName)
			return nil, errors.Wrapf(err, "error getting bucket %s", s3Config.BucketName)
		}
		if !ok {
			return nil, errors.Errorf("bucket %s does not exist", s3Config.BucketName)
		}
		c.s3Client = minioClient
	}

	if esConfig != nil {
		esClient, err := elasticsearch.NewClient(*esConfig)
		if err != nil {
			return nil, err
		}
		c.esClient = esClient
	}

	return c, nil
}

func (c *collector) GetMetadata(tr *tmv1beta1.Testrun) (*metadata.Metadata, error) {
	meta := metadata.FromTestrun(tr)
	components, err := componentdescriptor.GetComponentsFromLocations(tr)
	if err != nil {
		return nil, err
	}
	meta.ComponentDescriptor = components.JSON()
	return meta, nil
}

func (c *collector) Collect(tr *tmv1beta1.Testrun, metadata *metadata.Metadata) error {
	// only collect data of the right annotation is set
	if !metav1.HasAnnotation(tr.ObjectMeta, common.AnnotationCollectTestrun) {
		c.log.V(3).Info("skip result collection", "name", tr.Name, "namespace", tr.Namespace)
		return nil
	}

	// generate temporary result directory for downloaded artifacts
	tmpDir, err := ioutil.TempDir("", "collector")
	if err != nil {
		return errors.Wrapf(err, "unable to create cache directory for results")
	}

	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			c.log.Error(err, "unable to cleanup collector cache")
		}
	}()

	// add telemetry results to metadata
	//if telemetryData := c.getTelemetryResultsForRun(run); telemetryData != nil {
	//	run.Metadata.TelemetryData = telemetryData
	//}

	if err := c.collectSummaryAndExports(tmpDir, tr, metadata); err != nil {
		return err
	}

	if err := c.ingestIntoElasticsearch(tmpDir, tr); err != nil {
		c.log.Error(err, "error while ingesting file")
	}

	return nil
}
