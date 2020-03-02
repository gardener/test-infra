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

package result

import (
	"fmt"
	"github.com/Masterminds/semver"
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util/slack"
	"github.com/go-logr/logr"
	"github.com/olekukonko/tablewriter"
	"sort"
	"strings"
)

var SucessSymbols = map[bool]string{
	true:  "✅",
	false: "❌",
}

const NA = "N/A"

func (c *Collector) postTestrunsSummaryInSlack(config Config, log logr.Logger, runs testrunner.RunList) {
	if !config.PostSummaryInSlack {
		return
	}
	table, err := renderTableOfRuns(log, runs)
	if err != nil {
		log.Error(err, "failed creating a table to post")
	}
	if table == "" {
		log.Info("no table to render")
		return
	}

	slackClient, err := slack.New(log, config.SlackToken)
	if err != nil {
		log.Error(err, "Was not able to create slack client")
	}

	concourseURLFooter := ""
	if config.ConcourseURL != "" {
		concourseURLFooter = fmt.Sprintf("\nConcourse Job: %s", config.ConcourseURL)
	}

	if err := slackClient.PostMessage(config.SlackChannel, fmt.Sprintf("```%s\n%s\n%s```%s", header(), table, legend(), concourseURLFooter)); err != nil {
		log.Error(err, "failed to post the slack message of test summary")
	}
}

func header() string {
	return "Integration Test Results:"
}

func legend() string {
	return fmt.Sprintf(`
%s: Tests succeeded | %s: Tests failed | %s: Tests not applicable
`, SucessSymbols[true], SucessSymbols[false], NA)
}
func renderTableOfRuns(log logr.Logger, runs testrunner.RunList) (string, error) {
	writer := &strings.Builder{}
	table := tablewriter.NewWriter(writer)
	headerKeys := make(map[string]int, 0) // maps the header values to their index
	header := []string{""}

	for _, run := range runs {
		meta := run.Metadata
		if meta.CloudProvider != "" {
			header = append(header, meta.CloudProvider)
			headerKeys[meta.CloudProvider] = len(header) - 1
		}
	}

	res := results{
		header:  headerKeys,
		content: make(map[string]resultRow),
	}

	for _, run := range runs {
		meta := run.Metadata
		if meta.CloudProvider == "" {
			log.V(5).Info("skipped testrun", "id", meta.Testrun.ID)
			continue
		}

		dimensionKey := fmt.Sprintf("%s %s", meta.KubernetesVersion, meta.OperatingSystem)
		if meta.FlavorDescription != "" {
			dimensionKey = fmt.Sprintf("%s (%s)", dimensionKey, meta.FlavorDescription)
		}
		res.AddResult(meta, run.Testrun.Status.Phase == argov1.NodeSucceeded)
	}
	if res.Len() == 0 {
		return "", nil
	}

	table.SetHeader(header)
	table.AppendBulk(res.GetContent())
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.Render()
	return writer.String(), nil
}

type resultRow struct {
	dimension *metadata.Metadata
	content   []string
}

type results struct {
	header  map[string]int
	content map[string]resultRow
}

func (r *results) AddResult(meta *metadata.Metadata, success bool) {
	// should never happen but skip to ensure no panic
	_, ok := r.header[meta.CloudProvider]
	if !ok {
		return
	}
	key := computeDimensionKey(meta)
	if _, ok := r.content[key]; !ok {
		content := make([]string, len(r.header)+1)
		content[0] = key
		for i := 1; i < len(content); i++ {
			content[i] = NA
		}
		r.content[key] = resultRow{
			dimension: meta,
			content:   content,
		}
	}
	r.content[key].content[r.header[meta.CloudProvider]] = SucessSymbols[success]
}

func (r *results) GetContent() [][]string {
	rows := make(resultRows, len(r.content))

	i := 0
	for _, row := range r.content {
		rows[i] = row
		i++
	}
	sort.Sort(rows)
	return rows.GetContent()
}

func (r *results) Len() int {
	return len(r.content)
}

type resultRows []resultRow

func (l resultRows) GetContent() [][]string {
	content := make([][]string, len(l))
	for i, c := range l {
		content[i] = c.content
	}
	return content
}
func (l resultRows) Len() int      { return len(l) }
func (l resultRows) Swap(a, b int) { l[a], l[b] = l[b], l[a] }
func (l resultRows) Less(a, b int) bool {
	// sort by operating system name
	if l[a].dimension.OperatingSystem != l[b].dimension.OperatingSystem {
		return l[a].dimension.OperatingSystem < l[b].dimension.OperatingSystem
	}

	// sort by k8s version
	vA, err := semver.NewVersion(l[a].dimension.KubernetesVersion)
	if err != nil {
		return true
	}
	vB, err := semver.NewVersion(l[b].dimension.KubernetesVersion)
	if err != nil {
		return false
	}

	return vA.GreaterThan(vB)
}

func computeDimensionKey(meta *metadata.Metadata) string {
	dimensionKey := fmt.Sprintf("%s %s", meta.KubernetesVersion, meta.OperatingSystem)
	if meta.FlavorDescription != "" {
		dimensionKey = fmt.Sprintf("%s (%s)", dimensionKey, meta.FlavorDescription)
	}
	return dimensionKey
}
