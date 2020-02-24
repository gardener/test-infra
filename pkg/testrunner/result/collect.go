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
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	telemetrycommon "github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	trerrors "github.com/gardener/test-infra/pkg/testrunner/error"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"math"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Collect collects results of all testruns and writes them to a file.
// It returns whether there are failed testruns or not.
func (c *Collector) Collect(log logr.Logger, tmClient client.Client, namespace string, runs testrunner.RunList) (bool, error) {
	var (
		testrunsFailed = false
		result         *multierror.Error
	)

	c.fetchTelemetryResults()
	for _, run := range runs {
		runLogger := log.WithValues("testrun", run.Testrun.Name, "namespace", run.Testrun.Namespace)
		// Do only try to collect testruns results of testruns that ran into a timeout.
		// Any other error can not be retrieved.
		if run.Error != nil && !trerrors.IsTimeout(run.Error) {
			continue
		}

		// add telemetry results to metadata
		if telemetryData := c.getTelemetryResultsForRun(run); telemetryData != nil {
			run.Metadata.TelemetryData = telemetryData
		}

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

	c.uploadStatusAssets(c.config, log, runs, tmClient, log)
	fmt.Println(runs.RenderTable())

	return testrunsFailed, util.ReturnMultiError(result)
}

func getComponentsForUpload(cfg Config, runLogger logr.Logger) []*componentdescriptor.Component {
	componentsFromFile, err := componentdescriptor.GetComponentsFromFile(cfg.ComponentDescriptorPath)
	if err != nil {
		runLogger.Error(err, fmt.Sprintf("Unable to get component '%s'", cfg.ComponentDescriptorPath))
	}
	var componentsForUpload []*componentdescriptor.Component
	for _, componentName := range cfg.AssetComponents {
		if component := componentsFromFile.Get(componentName); component == nil {
			runLogger.Error(err, "can't find component", "component", cfg.AssetComponents)
		} else {
			componentsForUpload = append(componentsForUpload, component)
		}
	}
	return componentsForUpload
}

func (c *Collector) uploadStatusAssets(cfg Config, runLogger logr.Logger, runs testrunner.RunList, tmClient client.Client, log logr.Logger) {
	if !cfg.UploadStatusAsset {
		return
	}

	if len(cfg.AssetComponents) == 0 || cfg.GithubPassword == "" || cfg.GithubUser == "" || cfg.ComponentDescriptorPath == "" {
		err := errors.New("missing github password / github user / component descriptor path argument")
		log.Error(err, fmt.Sprintf("components: %s, ghUser: %s, ghPasswordLength: %d", cfg.AssetComponents, cfg.GithubUser, len(cfg.GithubPassword)))
		return
	}

	componentsForUpload := getComponentsForUpload(cfg, runLogger)
	if err := UploadStatusToGithub(log.WithName("github-upload"), runs, componentsForUpload, cfg.GithubUser, cfg.GithubPassword, cfg.AssetPrefix); err == nil {
		if err := MarkTestrunsAsUploadedToGithub(runLogger, tmClient, runs); err != nil {
			runLogger.Error(err, "unable to mark testrun status as uploaded to github")
		}
	}
	return
}

func (c *Collector) fetchTelemetryResults() {
	if c.telemetry != nil {
		c.log.Info("fetch telemetry controller summaryPath")
		_, figures, err := c.telemetry.StopAndAnalyze("", "text")
		if err != nil {
			c.log.Error(err, "unable to fetch telemetry measurements")
			return
		}
		c.telemetryResults = figures
	}
}

func (c *Collector) getTelemetryResultsForRun(run *testrunner.Run) *metadata.TelemetryData {
	if c.telemetry != nil && c.telemetryResults != nil {
		var shoot *common.ExtendedShoot
		switch s := run.Info.(type) {
		case *common.ExtendedShoot:
			shoot = s
		default:
			return nil
		}
		shootKey := telemetrycommon.GetShootKey(shoot.Name, shoot.Namespace)
		figure, ok := c.telemetryResults[shootKey]
		if !ok {
			return nil
		}

		dat := &metadata.TelemetryData{}
		if figure.ResponseTimeDuration != nil {
			dat.ResponseTime = &metadata.TelemetryResponseTimeDuration{
				Min:    figure.ResponseTimeDuration.Min,
				Max:    figure.ResponseTimeDuration.Max,
				Avg:    int64(math.Round(figure.ResponseTimeDuration.Avg)),
				Median: int64(math.Round(figure.ResponseTimeDuration.Median)),
				Std:    int64(math.Round(figure.ResponseTimeDuration.Std)),
			}
			dat.DowntimePeriods = &metadata.TelemetryDowntimePeriods{}
		}
		if figure.DownPeriods != nil {
			dat.DowntimePeriods = &metadata.TelemetryDowntimePeriods{
				Min:    int64(math.Round(figure.DownPeriods.Min)),
				Max:    int64(math.Round(figure.DownPeriods.Max)),
				Avg:    int64(math.Round(figure.DownPeriods.Avg)),
				Median: int64(math.Round(figure.DownPeriods.Median)),
				Std:    int64(math.Round(figure.DownPeriods.Std)),
			}
		}

		return dat
	}
	return nil
}
