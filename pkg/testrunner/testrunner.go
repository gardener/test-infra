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
	"fmt"
	"github.com/go-logr/logr"
	"sync"
	"time"

	trerrors "github.com/gardener/test-infra/pkg/testrunner/error"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/util"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var (
	maxWaitTime  = 1 * time.Hour
	pollInterval = 1 * time.Minute
)

// ExecuteTestruns deploys it to a testmachinery cluster and waits for the testruns results
func ExecuteTestruns(log logr.Logger, config *Config, runs RunList, testrunNamePrefix string) {
	log.V(3).Info(fmt.Sprintf("Config: %+v", util.PrettyPrintStruct(config)))
	maxWaitTime = config.Timeout
	pollInterval = config.Interval

	runs.Run(log.WithValues("namespace", config.Namespace), config.Client, config.Namespace, testrunNamePrefix, config.FlakeAttempts)
}

// runChart deploys the testruns in parallel into the testmachinery and watches them for their completion
func (rl RunList) Run(log logr.Logger, tmClient kubernetes.Interface, namespace, testrunNamePrefix string, maxFlakeAttempts int) {
	var wg sync.WaitGroup
	for i := range rl {
		if rl[i].Error != nil {
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			for flakeAttempt := 0; flakeAttempt <= maxFlakeAttempts; flakeAttempt++ {
				tr, err := runTestrun(log, tmClient, rl[i].Testrun, namespace, testrunNamePrefix)
				if err != nil {
					log.Error(err, "unable to run testrun")

					if trerrors.IsTimeout(err) {
						rl[i].Testrun.Status.Phase = tmv1beta1.PhaseStatusTimeout
					}
				}
				if tr != nil {
					rl[i].Testrun = tr
					rl[i].Metadata.Testrun.ID = tr.Name
				}
				rl[i].Error = err

				if err == nil && tr.Status.Phase == tmv1beta1.PhaseStatusSuccess {
					if flakeAttempt != 0 {
						rl[i].Metadata.Flaked++
					}
					// testrun was successful, break retry loop
					break
				}
				if flakeAttempt != maxFlakeAttempts {
					// clean status and name of testrun if it's failed to ignore it, since a retry will be initiated
					tr.Status = tmv1beta1.TestrunStatus{}
					tr.Name = ""
				}
			}

		}(i)
	}
	wg.Wait()
	log.Info("All testruns completed.")
}
