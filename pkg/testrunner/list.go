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
	"github.com/gardener/test-infra/pkg/util"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"strings"
	"sync"
	"time"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/go-logr/logr"
	"github.com/olekukonko/tablewriter"
)

// GetTestruns returns all testruns of a RunList as testrun array
func (rl RunList) GetTestruns() []*tmv1beta1.Testrun {
	testruns := make([]*tmv1beta1.Testrun, len(rl))
	for i, run := range rl {
		if run != nil {
			testruns[i] = run.Testrun
		}
	}
	return testruns
}

// HasErrors checks whether one run in list is erroneous.
func (rl RunList) HasErrors() bool {
	for _, run := range rl {
		if run.Error != nil {
			return true
		}
	}
	return false
}

// Errors returns all errors of all testruns in this testrun
func (rl RunList) Errors() error {
	var res *multierror.Error
	for _, run := range rl {
		if run.Error != nil {
			res = multierror.Append(res, run.Error)
		}
	}
	return util.ReturnMultiError(res)
}

// runChart deploys the testruns in parallel into the testmachinery and watches them for their completion
func (rl RunList) Run(log logr.Logger, config *Config, testrunNamePrefix string, notify ...chan *Run) {
	runID := uuid.New().String()
	log.Info(fmt.Sprintf("Starting testruns execution group %s", runID))
	var wg sync.WaitGroup
	for i := range rl {
		if rl[i].Error != nil {
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			for attempt := 0; attempt <= config.FlakeAttempts; attempt++ {
				rl[i].SetRunID(runID)
				triggerRunEvent(notify, rl[i])
				rl[i].Exec(log, config, testrunNamePrefix)
				rl[i].Metadata.Retries = attempt

				if rl[i].Error == nil && rl[i].Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess {
					// testrun was successful, break retry loop
					return
				}
				if attempt < config.FlakeAttempts {
					// clean status and name of testrun if it's failed to ignore it, since a retry will be initiated
					log.Info(fmt.Sprintf("testrun failed, retry %d/%d. testrun", attempt+1, config.FlakeAttempts))

					newRun, err := rl[i].Rerenderer.Rerender(rl[i].Testrun)
					if err != nil {
						log.Error(err, "unable to rerender testrun")
						return
					}
					*rl[i] = *newRun
				}
			}

		}(i)
	}
	wg.Wait()
	log.Info("All testruns completed.")
}

// RenderStatusTableForTestruns renders a status table for multiple testruns.
func (rl RunList) RenderTable() string {
	writer := &strings.Builder{}
	table := tablewriter.NewWriter(writer)
	table.SetHeader([]string{"Dimension", "Testrun", "Test Name", "Step", "Phase", "Duration"})

	dimensions := make(map[string][][]string, 0)
	for _, run := range rl {
		// dimension header
		dimension := getDimensionFromMetadata(run.Metadata)
		if _, ok := dimensions[dimension]; !ok {
			dimensions[dimension] = make([][]string, 0)
		}

		// testrun header
		tr := run.Testrun
		name := tr.Name
		if run.Metadata.Retries != 0 {
			name = fmt.Sprintf("%s(%d)", name, run.Metadata.Retries)
		}
		if purpose, ok := tr.GetAnnotations()[common.AnnotationTestrunPurpose]; ok {
			name = fmt.Sprintf("%s\n(%s)", name, purpose)
		}
		dimensions[dimension] = append(dimensions[dimension], []string{"", name})

		for _, s := range tr.Status.Steps {
			d := time.Duration(s.Duration) * time.Second
			dimensions[dimension] = append(dimensions[dimension], []string{"", "", s.TestDefinition.Name, s.Position.Step, string(s.Phase), d.String()})
		}
	}

	for dim, value := range dimensions {
		table.Append([]string{dim})
		table.AppendBulk(value)
	}

	table.Render()
	return writer.String()
}

func triggerRunEvent(notifyChannels []chan *Run, run *Run) {
	for _, c := range notifyChannels {
		go func(c chan *Run) {
			c <- run
		}(c)
	}
}

func getDimensionFromMetadata(meta *Metadata) string {
	d := fmt.Sprintf("%s/%s/%s", meta.CloudProvider, meta.KubernetesVersion, meta.OperatingSystem)
	if meta.FlavorDescription != "" {
		d = fmt.Sprintf("%s\n(%s)", d, meta.FlavorDescription)
	}
	return d
}
