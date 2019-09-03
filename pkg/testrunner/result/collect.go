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
	trerrors "github.com/gardener/test-infra/pkg/testrunner/error"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"path/filepath"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
)

// Collect collects results of all testruns and writes them to a file.
// It returns whether there are failed testruns or not.
func Collect(log logr.Logger, config *Config, tmClient kubernetes.Interface, namespace string, runs testrunner.RunList) (bool, error) {
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

		cfg := *config
		cfg.OutputDir = filepath.Join(config.OutputDir, util.RandomString(3))
		err := Output(runLogger, &cfg, tmClient, namespace, run.Testrun, run.Metadata)
		if err != nil {
			result = multierror.Append(result, err)
			continue
		}

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

		if run.Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess {
			runLogger.Info("Testrun finished successfully")
		} else {
			testrunsFailed = true
			runLogger.Error(fmt.Errorf("Testrun failed with phase %s", run.Testrun.Status.Phase), "")
		}
		fmt.Print(util.PrettyPrintStruct(run.Testrun.Status))
		printStatusTable(run.Testrun.Status.Steps)
	}

	return testrunsFailed, util.ReturnMultiError(result)
}
