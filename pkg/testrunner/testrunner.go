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
	"sync"

	trerrors "github.com/gardener/test-infra/pkg/testrunner/error"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/util"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"

	log "github.com/sirupsen/logrus"
)

var (
	maxWaitTimeSeconds  int64 = 3600
	pollIntervalSeconds int64 = 60
)

// ExecuteTestruns deploys it to a testmachinery cluster and waits for the testruns results
func ExecuteTestruns(config *Config, runs RunList, testrunNamePrefix string) {
	log.Debugf("Config: %+v", util.PrettyPrintStruct(config))
	maxWaitTimeSeconds = config.Timeout
	pollIntervalSeconds = config.Interval

	runs.Run(config.TmClient, config.Namespace, testrunNamePrefix)
}

// runChart deploys the testruns in parallel into the testmachinery and watches them for their completion
func (rl RunList) Run(tmClient kubernetes.Interface, namespace, testrunNamePrefix string) {
	var wg sync.WaitGroup
	for i := range rl {
		if rl[i].Error != nil {
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			tr, err := runTestrun(tmClient, rl[i].Testrun, namespace, testrunNamePrefix)
			if err != nil {
				log.Error(err.Error())

				if trerrors.IsTimeout(err) {
					rl[i].Testrun.Status.Phase = tmv1beta1.PhaseStatusTimeout
				}
			}
			if tr != nil {
				rl[i].Testrun = tr
				rl[i].Metadata.Testrun.ID = tr.Name
			}
			rl[i].Error = err

		}(i)
	}
	wg.Wait()
	log.Infof("All testruns completed.")
}
