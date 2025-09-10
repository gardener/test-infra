// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package output

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/renderer"
	"github.com/olekukonko/tablewriter/tw"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// RenderTestflowTable creates a human-readable table of a testflow.
func RenderTestflowTable(writer io.Writer, flow tmv1beta1.TestFlow) {
	table := tablewriter.NewTable(writer,
		tablewriter.WithHeader([]string{"Step", "Definition", "Dependencies"}),
		tablewriter.WithHeaderAutoWrap(tw.WrapNormal),
		tablewriter.WithRowAutoWrap(tw.WrapNormal),
		tablewriter.WithRenderer(renderer.NewBlueprint()),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleASCII),
			Borders: tw.Border{Top: tw.On, Bottom: tw.On, Left: tw.On, Right: tw.On},
			Settings: tw.Settings{
				Separators: tw.Separators{
					BetweenRows: tw.On,
				},
			},
		}),
	)

	for _, s := range flow {
		definition := ""
		if s.Definition.Name != "" {
			definition = fmt.Sprintf("Name: %s", s.Definition.Name)
		}
		if s.Definition.Label != "" {
			definition = fmt.Sprintf("Label: %s", s.Definition.Label)
		}

		if err := table.Append([]string{s.Name, definition, strings.Join(s.DependsOn, "\n")}); err != nil {
			fmt.Fprintf(os.Stderr, "Could not append row to testflow table: %v", err)
		}
	}
	if err := table.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "Could not render testflow table: %v", err)
	}
}

// RenderStatusTable creates a human-readable table for testrun status steps.
// The steps are ordered by start time and step name.
func RenderStatusTable(writer io.Writer, steps []*tmv1beta1.StepStatus) {
	orderSteps(steps)

	table := tablewriter.NewTable(writer,
		tablewriter.WithHeader([]string{"Name", "Step", "Phase", "Duration"}),
		tablewriter.WithRenderer(renderer.NewBlueprint()),
		tablewriter.WithRendition(tw.Rendition{
			Symbols: tw.NewSymbols(tw.StyleASCII),
			Borders: tw.Border{Top: tw.On, Bottom: tw.On, Left: tw.On, Right: tw.On},
		}),
	)
	if err := table.Bulk(GetStatusTableRows(steps)); err != nil {
		fmt.Fprintf(os.Stderr, "Could not append rows to status table: %v", err)
	}
	if err := table.Render(); err != nil {
		fmt.Fprintf(os.Stderr, "Could not render status table: %v", err)
	}
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
	// order by step name if start date is not set
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
