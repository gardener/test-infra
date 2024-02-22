// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package docs

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/gardener/test-infra/pkg/logger"
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
