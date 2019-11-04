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

package util

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// RenderTestflowTable creates a human readable table of a testflow.
func RenderTestflowTable(writer io.Writer, flow tmv1beta1.TestFlow) {
	table := tablewriter.NewWriter(writer)
	table.SetHeader([]string{"Step", "Definition", "Dependencies"})
	table.SetAutoWrapText(true)
	table.SetRowSeparator("-")
	table.SetRowLine(true)

	for _, s := range flow {
		definition := ""
		if s.Definition.Name != "" {
			definition = fmt.Sprintf("Name: %s", s.Definition.Name)
		}
		if s.Definition.Label != "" {
			definition = fmt.Sprintf("Label: %s", s.Definition.Label)
		}

		table.Append([]string{s.Name, definition, strings.Join(s.DependsOn, "\n")})
	}
	table.Render()
}

// RenderStatusTableForTestruns renders a status table for multiple testruns.
func RenderStatusTableForTestruns(writer io.Writer, testruns []*tmv1beta1.Testrun) {
	table := tablewriter.NewWriter(writer)
	table.SetHeader([]string{"Testrun", "Test Name", "Step", "Phase", "Duration"})
	for _, tr := range testruns {
		trHeader := []string{tr.Name}
		table.Append(trHeader)
		for _, s := range tr.Status.Steps {
			d := time.Duration(s.Duration) * time.Second
			table.Append([]string{"", s.TestDefinition.Name, s.Position.Step, string(s.Phase), d.String()})
		}
	}
	table.Render()
}

// RenderStatusTable creates a human readable table for testrun status steps.
// The steps are ordered by starttime and step name.
func RenderStatusTable(writer io.Writer, steps []*tmv1beta1.StepStatus) {
	orderSteps(steps)

	table := tablewriter.NewWriter(writer)
	table.SetHeader([]string{"Name", "Step", "Phase", "Duration"})

	table.AppendBulk(GetStatusTableRows(steps))
	table.Render()
}

func GetStatusTableRows(steps []*tmv1beta1.StepStatus) [][]string {
	rows := make([][]string, len(steps))
	for i, s := range steps {
		d := time.Duration(s.Duration) * time.Second
		rows[i] = []string{s.TestDefinition.Name, s.Position.Step, string(s.Phase), d.String()}
	}
	return rows
}

// orderSteps orders the steps by their finished date.
// If the ddate is not defined the step status are ordered by their step name
func orderSteps(steps []*tmv1beta1.StepStatus) {
	sort.Sort(StepStatusList(steps))
}

type StepStatusList []*tmv1beta1.StepStatus

func (s StepStatusList) Less(a, b int) bool {
	// order by step name if startdate is not set
	if s[a].StartTime.IsZero() && s[b].StartTime.IsZero() {
		return s[a].Position.Step < s[b].Position.Step
	}
	if s[a].StartTime.IsZero() {
		return false
	}
	if s[b].StartTime.IsZero() {
		return true
	}
	return s[a].StartTime.Before(s[b].StartTime)
}
func (s StepStatusList) Len() int      { return len(s) }
func (s StepStatusList) Swap(a, b int) { s[a], s[b] = s[b], s[a] }
