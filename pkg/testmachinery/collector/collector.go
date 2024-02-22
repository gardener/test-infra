// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/apis/config"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util/elasticsearch"
	"github.com/gardener/test-infra/pkg/util/s3"
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

	esConfig *config.ElasticSearch
	esClient elasticsearch.Client
	s3Config *config.S3
	s3Client s3.Client
}

func New(log logr.Logger, k8sClient client.Client, esConfig *config.ElasticSearch, s3Config *config.S3) (Interface, error) {
	c := &collector{
		log:      log,
		client:   k8sClient,
		esConfig: esConfig,
		s3Config: s3Config,
	}

	if s3Config != nil {
		s3Client, err := s3.New(s3.FromConfig(s3Config))
		if err != nil {
			return nil, err
		}
		c.s3Client = s3Client
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
	tmpDir, err := os.MkdirTemp("", "collector")
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
