// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package output

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
