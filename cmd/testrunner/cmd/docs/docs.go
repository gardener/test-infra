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

package docs

import (
	"bytes"
	"fmt"
	"github.com/gardener/test-infra/pkg/logger"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var (
	outputDir string
)

func init() {
	docsCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Directory where the doc is written to.")
	if err := docsCmd.MarkFlagFilename("output"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "output")
	}
}

// AddCommand adds run-testrun to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(docsCmd)
}

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate docs for the testrunner",
	Run: func(cmd *cobra.Command, args []string) {
		if outputDir == "" {
			buf := getDoc(cmd.Parent())
			if _, err := os.Stdout.Write(buf.Bytes()); err != nil {
				logger.Log.Error(err, "unable to write output to stdout")
				os.Exit(1)
			}
			return
		}
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			logger.Log.Error(err, "cannot create directories")
			os.Exit(1)
		}
		cmd.Parent().DisableAutoGenTag = true
		err = doc.GenMarkdownTree(cmd.Parent(), outputDir)
		if err != nil {
			logger.Log.Error(err, "unable to create markdown")
			os.Exit(1)
		}
		logger.Log.Info(fmt.Sprintf("Successfully written docs to %s", outputDir))
	},
}

func getDoc(cmd *cobra.Command) *bytes.Buffer {
	buf := new(bytes.Buffer)
	cmd.DisableAutoGenTag = true

	err := doc.GenReST(cmd, buf)
	if err != nil {
		return buf
	}

	for _, c := range cmd.Commands() {
		cDoc := getDoc(c)
		buf.Write(cDoc.Bytes())
	}
	return buf
}
