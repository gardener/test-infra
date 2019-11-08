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
	"fmt"
	"github.com/gardener/test-infra/pkg/util/output"
	"os"
	"path/filepath"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	trerrors "github.com/gardener/test-infra/pkg/testrunner/error"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
)

// Collect collects results of all testruns and writes them to a file.
// It returns whether there are failed testruns or not.
func (c *Collector) Collect(log logr.Logger, tmClient kubernetes.Interface, namespace string, runs testrunner.RunList) (bool, error) {
	var (
		testrunsFailed = false
		result         *multierror.Error
	)
	for _, run := range runs {
		runLogger := log.WithValues("testrun", run.Testrun.Name, "namespace", run.Testrun.Namespace)
		// Do only try to collect testruns results of testruns that ran into a timeout.
		// Any other error can not be retrieved.
		if run.Error != nil && !trerrors.IsTimeout(run.Error) {
			continue
		}

		cfg := c.config
		cfg.OutputDir = filepath.Join(cfg.OutputDir, util.RandomString(3))
		err := Output(runLogger, &cfg, tmClient, namespace, run.Testrun, run.Metadata)
		if err != nil {
			result = multierror.Append(result, err)
			continue
		}

		c.ingestIntoElasticsearch(cfg, err, runLogger, tmClient, run)
		c.uploadStatusAsset(cfg, runLogger, err, run, tmClient)

		if run.Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess {
			runLogger.Info("Testrun finished successfully")
		} else {
			testrunsFailed = true
			runLogger.Error(fmt.Errorf("Testrun failed with phase %s", run.Testrun.Status.Phase), "")
		}
		fmt.Print(util.PrettyPrintStruct(run.Testrun.Status))
		output.RenderStatusTable(os.Stdout, run.Testrun.Status.Steps)
		fmt.Println(":---------------------------------------------------------------------------------------------:")
	}

	output.RenderStatusTableForTestruns(os.Stdout, runs)

	c.fetchTelemetryResults()

	return testrunsFailed, util.ReturnMultiError(result)
}

func (c *Collector) ingestIntoElasticsearch(cfg Config, err error, runLogger logr.Logger, tmClient kubernetes.Interface, run *testrunner.Run) {
	if cfg.OutputDir != "" && cfg.ESConfigName != "" {
		err = IngestDir(runLogger, cfg.OutputDir, cfg.ESConfigName)
		if err != nil {
			runLogger.Error(err, "cannot persist file", "file", cfg.OutputDir)
		} else {
			err := MarkTestrunsAsIngested(runLogger, tmClient, run.Testrun)
			if err != nil {
				runLogger.Error(err, "unable to ingest testrun")
			}
		}
	}
}

func (c *Collector) uploadStatusAsset(cfg Config, runLogger logr.Logger, err error, run *testrunner.Run, tmClient kubernetes.Interface) {
	if !cfg.UploadStatusAsset {
		return
	}

	if len(cfg.AssetComponents) == 0 || cfg.GithubPassword == "" || cfg.GithubUser == "" || cfg.ComponentDescriptorPath == "" {
		runLogger.Error(err, "missing github password / github user / component descriptor path argument")
	}
	componentsFromFile, err := componentdescriptor.GetComponentsFromFile(cfg.ComponentDescriptorPath)
	if err != nil {
		runLogger.Error(err, fmt.Sprintf("Unable to get component '%s'", cfg.ComponentDescriptorPath))
	}

	assetUploadSuccessful := true
	var componentsForUpload []*componentdescriptor.Component
	for _, componentName := range cfg.AssetComponents {
		if component := componentsFromFile.Get(componentName); component == nil {
			runLogger.Error(err, "can't find component", "component", cfg.AssetComponents)
			assetUploadSuccessful = false
		} else {
			componentsForUpload = append(componentsForUpload, component)
		}
	}
	for _, component := range componentsForUpload {
		if err := UploadStatusToGithub(runLogger, run, component, cfg.GithubUser, cfg.GithubPassword, cfg.AssetPrefix); err != nil {
			runLogger.Error(err, "unable to attach testrun status to github component")
			assetUploadSuccessful = false
		}
	}
	if assetUploadSuccessful {
		err := MarkTestrunsAsUploadedToGithub(runLogger, tmClient, run.Testrun)
		if err != nil {
			runLogger.Error(err, "unable to mark testrun status as uploaded to github")
		}
	}
}

func (c *Collector) fetchTelemetryResults() {
	if c.telemetry != nil {
		c.log.Info("fetch telemetry controller summaryPath")
		_, err := c.telemetry.StopAndAnalyze("", "text")
		if err != nil {
			c.log.Error(err, "unable to fetch telemetry measurements")
			return
		}
	}
}
