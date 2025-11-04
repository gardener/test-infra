// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package run_testrun

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
)

func NewRunTestrunCommand() (*cobra.Command, error) {
	opts := NewOptions()

	cmd := &cobra.Command{
		Use:   "run-testrun",
		Short: "Run the testrunner with a testrun",
		Aliases: []string{
			"run-tr",
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Validate()
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancelFunc := context.WithTimeout(context.Background(), opts.testrunnerConfig.Timeout)
			defer cancelFunc()

			if err := opts.run(ctx); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}

	if err := opts.AddFlags(cmd.Flags()); err != nil {
		return nil, err
	}

	return cmd, nil
}

func (o *options) run(ctx context.Context) error {
	logger.Log.Info("start testmachinery testrunner")

	logger.InitializeSummarySetup(o.summaryFilePath)

	watcher, err := watch.NewFromFile(logger.Log, o.tmKubeconfigPath, &o.watchOptions)
	if err != nil {
		logger.Log.Error(err, "unable to start testrun watch controller")
		os.Exit(1)
	}

	go func() {
		if err := watcher.Start(ctx); err != nil {
			logger.Log.Error(err, "unable to start testrun watch controller")
			os.Exit(1)
		}
	}()

	if err := watch.WaitForCacheSyncWithTimeout(watcher, 2*time.Minute); err != nil {
		return err
	}
	o.testrunnerConfig.Watch = watcher

	tr, err := testmachinery.ParseTestrunFromFile(o.testrunPath)
	if err != nil {
		logger.Log.Error(err, "unable to parse testrun")
		os.Exit(1)
	}

	run := testrunner.Run{
		Testrun:  tr,
		Metadata: &metadata.Metadata{},
	}

	run.Exec(logger.Log.WithName("execute"), &o.testrunnerConfig, o.testrunNamePrefix)
	if run.Error != nil {
		logger.Log.Error(run.Error, "testrunner execution disrupted")
		os.Exit(1)
	}

	if run.Testrun.Status.Phase == tmv1beta1.RunPhaseSuccess {
		logger.Log.Info("testrunner successfully finished.")
	} else {
		logger.Log.Error(nil, "testrunner finished unsuccessful", "phase", run.Testrun.Status.Phase)
	}

	fmt.Print(util.PrettyPrintStruct(run.Testrun.Status))

	return nil
}
