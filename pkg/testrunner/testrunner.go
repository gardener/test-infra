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
	"errors"
	"sync"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/util"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"

	log "github.com/sirupsen/logrus"
)

var (
	maxWaitTimeSeconds  int64 = 3600
	pollIntervalSeconds int64 = 60
)

// ExecuteTestrun deploys it to a testmachinery cluster and waits for the testruns results
func ExecuteTestrun(config *Config, runs RunList, testrunNamePrefix string) (RunList, error) {
	log.Debugf("Config: %+v", util.PrettyPrintStruct(config))
	maxWaitTimeSeconds = config.Timeout
	pollIntervalSeconds = config.Interval

	finishedTestruns := runChart(config.TmClient, runs, config.Namespace, testrunNamePrefix)

	if len(finishedTestruns) == 0 {
		return nil, errors.New("No testruns finished")
	}

	return finishedTestruns, nil
}

// runChart deploys the testruns in parallel into the testmachinery and watches them for their completion
func runChart(tmClient kubernetes.Interface, runs RunList, namespace, testrunNamePrefix string) RunList {
	var wg sync.WaitGroup
	mutex := &sync.Mutex{}
	finishedTestruns := RunList{}
	for _, r := range runs {
		run := *r

		wg.Add(1)
		go func(r *Run) {
			defer wg.Done()
			tr, err := runTestrun(tmClient, r.Testrun, namespace, testrunNamePrefix)
			if err != nil {
				log.Error(err.Error())
				if tr != nil {
					r.Testrun = tr
				}
				r.Testrun.Status.Phase = tmv1beta1.PhaseStatusError
			} else {
				r.Testrun = tr
				r.Metadata.Testrun.ID = tr.Name
			}
			mutex.Lock()
			finishedTestruns = append(finishedTestruns, r)
			mutex.Unlock()
		}(&run)
	}
	wg.Wait()
	log.Infof("All testruns completed.")
	return finishedTestruns
}
