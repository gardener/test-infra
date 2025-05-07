// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/gardener/test-infra/cmd/testrunner/cmd/alert"
	collectcmd "github.com/gardener/test-infra/cmd/testrunner/cmd/collect"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/docs"
	notifycmd "github.com/gardener/test-infra/cmd/testrunner/cmd/notify"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/run_template"
	"github.com/gardener/test-infra/cmd/testrunner/cmd/run_testrun"
	versioncmd "github.com/gardener/test-infra/cmd/testrunner/cmd/version"
	"github.com/gardener/test-infra/pkg/logger"
)

var rootCmd = &cobra.Command{
	Use:   "testrunner",
	Short: "Testrunner for Test Machinery",
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		log, err := logger.NewCliLogger()
		if err != nil {
			return err
		}
		logger.SetLogger(log)
		ctrl.SetLogger(log)
		return nil
	},
}

// Execute executes the testrunner cli commands
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
}

func init() {
	logger.InitFlags(rootCmd.PersistentFlags())
	rootCmd.PersistentFlags().Bool("dry-run", false, "Dry run will print the rendered template")

	alert.AddCommand(rootCmd)
	addCommand(run_template.NewRunTemplateCommand)
	addCommand(run_testrun.NewRunTestrunCommand)
	collectcmd.AddCommand(rootCmd)
	notifycmd.AddCommand(rootCmd)
	docs.AddCommand(rootCmd)
	versioncmd.AddCommand(rootCmd)
}

func addCommand(add func() (*cobra.Command, error)) {
	cmd, err := add()
	if err != nil {
		panic(err)
	}
	rootCmd.AddCommand(cmd)
}
