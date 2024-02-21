// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package versioncmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/version"
)

// AddCommand adds version to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "GetInterface testrunner version",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Log.V(3).Info("version")
		v, err := json.Marshal(version.Get())
		if err != nil {
			logger.Log.Error(err, "unable to marshal version")
			os.Exit(1)
		}
		fmt.Print(string(v))
	},
}
