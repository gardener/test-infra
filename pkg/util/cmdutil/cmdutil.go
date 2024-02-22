// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmdutil

import (
	"io"

	"github.com/olekukonko/tablewriter"
)

func PrintTable(output io.Writer, headers []string, content [][]string) {
	table := tablewriter.NewWriter(output)
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetCenterSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderAlignment(3)
	table.SetAlignment(3)
	table.SetHeaderLine(false)

	table.SetHeader(headers)
	table.AppendBulk(content)
	table.Render()
}
