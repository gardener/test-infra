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
