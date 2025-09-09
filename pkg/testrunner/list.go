// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testrunner

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/util"
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
func (rl RunList) Run(log logr.Logger, config *Config, testrunNamePrefix string, notify ...chan *Run) error {
	var executiongroupID string
	var tmDashboardURL string
	if !config.NoExecutionGroup {
		executiongroupID = uuid.New().String()
		// Print dashboard url if possible and if a execution group is defined
		log.Info(fmt.Sprintf("Starting testruns execution group %s", executiongroupID))
		TMDashboardHost, err := GetTMDashboardHost(config.Watch.Client())
		if err != nil {
			log.V(3).Info("unable to get TestMachinery Dashboard URL", "error", err.Error())
		} else {
			tmDashboardURL = GetTmDashboardURLFromHostForExecutionGroup(TMDashboardHost, executiongroupID)
			log.Info(fmt.Sprintf("TestMachinery Dashboard: %s", tmDashboardURL))
		}
	}

	executor, err := NewExecutor(log, config.ExecutorConfig)
	if err != nil {
		return err
	}

	for i := range rl {
		if rl[i].Error != nil {
			continue
		}

		var (
			trI     = i
			attempt = 0
			f       func()
		)
		f = func() {
			rl[trI].SetRunID(executiongroupID)
			rl[trI].SetTMDashboardURL(tmDashboardURL)
			triggerRunEvent(notify, rl[trI])
			rl[trI].Exec(log, config, testrunNamePrefix)
			if rl[trI].Metadata != nil {
				rl[trI].Metadata.Retries = attempt
			}

			if rl[trI].Error == nil && rl[trI].Testrun.Status.Phase == tmv1beta1.RunPhaseSuccess {
				// testrun was successful, break retry loop
				return
			}
			if attempt == config.FlakeAttempts {
				return
			}

			// retry the testrun

			// clean status and name of testrun if it's failed to ignore it, since a retry will be initiated
			log.Info(fmt.Sprintf("testrun failed, retry %d/%d. testrun", attempt+1, config.FlakeAttempts))

			newRun, err := rl[trI].Rerenderer.Rerender(rl[trI].Testrun)
			if err != nil {
				log.Error(err, "unable to rerender testrun")
				return
			}

			attempt++

			// update retry metadata and annotation
			newRun.Metadata.Retries = attempt
			newRun.Testrun.Annotations[common.AnnotationRetries] = strconv.Itoa(attempt)
			newRun.Testrun.Annotations[common.AnnotationPreviousAttempt] = rl[trI].Testrun.Name

			*rl[trI] = *newRun
			executor.AddItem(f)
		}
		executor.AddItem(f)
	}

	executor.Run()

	log.Info("All testruns completed.")
	return nil
}

// RenderStatusTableForTestruns renders a status table for multiple testruns.
func (rl RunList) RenderTable() string {
	writer := &strings.Builder{}
	table := tablewriter.NewTable(writer,
		tablewriter.WithHeader([]string{"Dimension", "Testrun", "Test Name", "Step", "Phase", "Duration"}),
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
	)

	dimensions := make(map[string][][]string)
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

		util.OrderStepsStatus(tr.Status.Steps)
		for _, s := range tr.Status.Steps {
			d := time.Duration(s.Duration) * time.Second
			dimensions[dimension] = append(dimensions[dimension], []string{"", "", s.TestDefinition.Name, s.Position.Step, string(s.Phase), d.String()})
		}
	}

	for dim, value := range dimensions {
		if err := table.Append([]string{dim}); err != nil {
			return fmt.Errorf("unable to add table content: %w", err).Error()
		}
		if err := table.Bulk(value); err != nil {
			return fmt.Errorf("unable to add table content: %w", err).Error()
		}
	}

	if err := table.Render(); err != nil {
		return fmt.Errorf("unable to render table: %w", err).Error()
	}
	return writer.String()
}

func triggerRunEvent(notifyChannels []chan *Run, run *Run) {
	for _, c := range notifyChannels {
		select {
		case c <- run:
		default:
		}
	}
}

func getDimensionFromMetadata(meta *metadata.Metadata) string {
	d := fmt.Sprintf("%s/%s/%s", meta.CloudProvider, meta.KubernetesVersion, meta.OperatingSystem)
	if meta.FlavorDescription != "" {
		d = fmt.Sprintf("%s\n(%s)", d, meta.FlavorDescription)
	}
	return d
}
