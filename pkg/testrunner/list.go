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
	"strings"
	"sync"
	"time"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/go-logr/logr"
	"github.com/olekukonko/tablewriter"
)

// runChart deploys the testruns in parallel into the testmachinery and watches them for their completion
func (rl RunList) Run(log logr.Logger, config *Config, testrunNamePrefix string) {
	var wg sync.WaitGroup
	for i := range rl {
		if rl[i].Error != nil {
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			for attempt := 0; attempt <= config.FlakeAttempts; attempt++ {
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
		if purpose, ok := tr.GetAnnotations()[common.PurposeTestrunAnnotation]; ok {
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

func getDimensionFromMetadata(meta *Metadata) string {
	d := fmt.Sprintf("%s/%s/%s", meta.CloudProvider, meta.KubernetesVersion, meta.OperatingSystem)
	if meta.FlavorDescription != "" {
		d = fmt.Sprintf("%s\n(%s)", d, meta.FlavorDescription)
	}
	return d
}
