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
	"fmt"
	"sync"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"

	"k8s.io/client-go/tools/clientcmd"

	tmclientset "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned"
	log "github.com/sirupsen/logrus"
)

var (
	maxWaitTimeSeconds  int64 = 3600
	pollIntervalSeconds int64 = 60
)

// Run renders a testrun, deploys it to a testmachinery cluster and waits for the testruns results
func Run(config *Config, testruns []*tmv1beta1.Testrun, testrunNamePrefix string) ([]*tmv1beta1.Testrun, error) {
	log.Debugf("Config: %+v", util.PrettyPrintStruct(config))
	maxWaitTimeSeconds = config.Timeout
	pollIntervalSeconds = config.Interval

	tmConfig, err := clientcmd.BuildConfigFromFlags("", config.TmKubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("Cannot build kubernetes client from %s: %s", config.TmKubeconfigPath, err.Error())
	}
	tmClient, err := tmclientset.NewForConfig(tmConfig)
	if err != nil {
		return nil, err
	}

	finishedTestruns := runChart(tmClient, testruns, config.Namespace, testrunNamePrefix)

	if len(finishedTestruns) == 0 {
		return nil, errors.New("No testruns finished")
	}

	return finishedTestruns, nil
}

// runChart tries to parse each rendered file of a chart into a testrun.
// If a filecontent is a testrun then it is deployed into the testmachinery.
func runChart(tmClient *tmclientset.Clientset, testruns []*tmv1beta1.Testrun, namespace, testrunNamePrefix string) []*tmv1beta1.Testrun {
	var wg sync.WaitGroup
	mutex := &sync.Mutex{}
	finishedTestruns := []*tmv1beta1.Testrun{}
	for _, tr := range testruns {
		testrun := *tr

		wg.Add(1)
		go func(tr *tmv1beta1.Testrun) {
			defer wg.Done()
			tr, err := runTestrun(tmClient, tr, namespace, testrunNamePrefix)
			if err != nil {
				log.Error(err.Error())
				tr.Status.Phase = tmv1beta1.PhaseStatusFailed
			}
			mutex.Lock()
			finishedTestruns = append(finishedTestruns, tr)
			mutex.Unlock()
		}(&testrun)
	}
	wg.Wait()
	log.Infof("All testruns completed.")
	return finishedTestruns
}
