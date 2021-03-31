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
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/go-logr/logr"
	"github.com/olekukonko/tablewriter"
)

// SymbolOffset is the offset that the symbol is prefixed for better readability
const SymbolOffset = " "

// StatusSymbol is a unicode symbol for dieplaying in a table
type StatusSymbol string

const (
	StatusSymbolSuccess StatusSymbol = "✅"
	StatusSymbolFailure StatusSymbol = "❌"
	StatusSymbolNA      StatusSymbol = "N/A"
	StatusSymbolError   StatusSymbol = "🔥"
	StatusSymbolUnknown StatusSymbol = "❓"
)

type TableItems []*TableItem

type TableItem struct {
	Meta         ItemMeta
	StatusSymbol StatusSymbol
}

type ItemMeta struct {
	CloudProvider           string
	TestrunID               string
	OperatingSystem         string
	KubernetesVersion       string
	FlavorDescription       string
	AdditionalDimensionInfo string
}

type resultRow struct {
	dimension ItemMeta
	content   []string
}

type results struct {
	header  map[string]int
	content map[string]resultRow
}

// RenderTableForSlack renders a string table of given table items
func RenderTableForSlack(log logr.Logger, items TableItems) (string, error) {
	writer := &strings.Builder{}
	table := tablewriter.NewWriter(writer)
	headerKeys := make(map[string]int) // maps the header values to their index
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
		res.AddResult(meta, item.StatusSymbol)
	}
	if res.Len() == 0 {
		return "", nil
	}

	table.SetAutoWrapText(false)
	table.SetHeader(header)
	table.AppendBulk(res.GetContent())
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.Render()
	return writer.String(), nil
}

func (r *results) AddResult(meta ItemMeta, symbol StatusSymbol) {
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
			content[i] = SymbolOffset + string(StatusSymbolNA)
		}
		r.content[key] = resultRow{
			dimension: meta,
			content:   content,
		}
	}
	r.content[key].content[cpIndex] = SymbolOffset + string(symbol)
}

func computeDimensionKey(meta ItemMeta) string {
	dimensionKey := fmt.Sprintf("%s %s", meta.KubernetesVersion, meta.OperatingSystem)
	if meta.AdditionalDimensionInfo != "" {
		dimensionKey = fmt.Sprintf("%s [%s]", dimensionKey, meta.AdditionalDimensionInfo)
	}
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

// SplitString splits a string into byte slices of the given length.
// Tt will try to split only on newlines.
func SplitString(text string, size int) []string {
	if len(text) < size {
		return []string{text}
	}

	textByNL := strings.SplitAfter(text, "\n")
	if len(textByNL) == 1 {
		return splitStringWithSize(text, size)
	}
	var (
		chunk string
	)
	chunks := make([]string, 0)
	for _, text := range textByNL {
		if (len(chunk) + len(text)) >= size {
			chunks = append(chunks, chunk)
			chunk = ""
		}
		if len(text) > size {
			chunks = append(chunks, splitStringWithSize(text, size)...)
			continue
		}
		chunk = chunk + text
	}
	if len(chunk) > 0 {
		chunks = append(chunks, chunk)
	}

	return chunks
}

func splitStringWithSize(data string, size int) []string {
	if len(data) < size {
		return []string{data}
	}

	var chunk string
	chunks := make([]string, 0, len(data)/size+1)
	for len(data) >= size {
		chunk, data = data[:size], data[size:]
		chunks = append(chunks, chunk)
	}
	if len(data) > 0 {
		chunks = append(chunks, data)
	}
	return chunks
}
