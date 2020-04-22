// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
	"github.com/Masterminds/semver"
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

//
type resultRow struct {
	dimension ItemMeta
	content   []string
}

type results struct {
	header  map[string]int
	content map[string]resultRow
}

type TableItems []*TableItem

type TableItem struct {
	Meta    ItemMeta
	Success bool
}

type ItemMeta struct {
	CloudProvider     string
	TestrunID         string
	OperatingSystem   string
	KubernetesVersion string
	FlavorDescription string
}

// RenderTableForSlack renders a string table of given table items
func RenderTableForSlack(log logr.Logger, items TableItems) (string, error) {
	writer := &strings.Builder{}
	table := tablewriter.NewWriter(writer)
	headerKeys := make(map[string]int, 0) // maps the header values to their index
	header := []string{""}

	for _, item := range items {
		if _, ok := headerKeys[item.Meta.CloudProvider]; !ok {
			header = append(header, item.Meta.CloudProvider)
			headerKeys[item.Meta.CloudProvider] = len(header) - 1
		}
	}

	res := results{
		header:  headerKeys,
		content: make(map[string]resultRow),
	}

	for _, item := range items {
		meta := item.Meta
		if meta.CloudProvider == "" {
			log.V(5).Info("skipped testrun", "id", meta.TestrunID)
			continue
		}

		dimensionKey := fmt.Sprintf("%s %s", meta.KubernetesVersion, meta.OperatingSystem)
		if meta.FlavorDescription != "" {
			dimensionKey = fmt.Sprintf("%s (%s)", dimensionKey, meta.FlavorDescription)
		}
		res.AddResult(meta, item.Success)
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

func (r *results) AddResult(meta ItemMeta, success bool) {
	// should never happen but skip to ensure no panic
	cpIndex, ok := r.header[meta.CloudProvider]
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
	r.content[key].content[cpIndex] = SucessSymbols[success]
}

func computeDimensionKey(meta ItemMeta) string {
	dimensionKey := fmt.Sprintf("%s %s", meta.KubernetesVersion, meta.OperatingSystem)
	if meta.FlavorDescription != "" {
		dimensionKey = fmt.Sprintf("%s (%s)", dimensionKey, meta.FlavorDescription)
	}
	return dimensionKey
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
