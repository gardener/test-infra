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
