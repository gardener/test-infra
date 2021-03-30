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

package completion

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd *cobra.Command

func RegisterCmd(cmd *cobra.Command) {
	rootCmd = cmd
	cmd.AddCommand(completionCmd)
}

// completionCmd represents the completion command
var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generates bash or zsh completion scripts",
	Long: `To load completion run

. <(hostscheduler completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(hostscheduler completion)
`,
	ValidArgs: []string{"bash", "zsh"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 || args[0] == "bash" {
			if err := rootCmd.GenBashCompletion(os.Stdout); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		} else if args[0] == "zsh" {
			if err := rootCmd.GenZshCompletion(os.Stdout); err != nil {
				fmt.Printf("Unbale to generate zsh completion\n%s\n", err.Error())
				os.Exit(1)
			}
		}
	},
}
