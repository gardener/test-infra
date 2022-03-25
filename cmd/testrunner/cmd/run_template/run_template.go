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

package run_template

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"

	"github.com/gardener/test-infra/pkg/logger"

	"github.com/gardener/test-infra/pkg/util"

	"github.com/spf13/cobra"

	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	testrunnerTemplate "github.com/gardener/test-infra/pkg/testrunner/template"
)

// NewRunTemplateCommand creates a new run template command.
func NewRunTemplateCommand() (*cobra.Command, error) {
	opts := NewOptions()

	cmd := &cobra.Command{
		Use:   "run-template",
		Short: "Run the testrunner with a helm template containing testruns",
		Aliases: []string{
			"run", // for backward compatibility
			"run-tmpl",
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return opts.Validate()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

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

// run runs the workflow for the run template
func (o *options) run(ctx context.Context) error {
	logger.Log.Info("Start testmachinery testrunner")

	runs, err := testrunnerTemplate.RenderTestruns(ctx, logger.Log.WithName("Render"), &o.shootParameters, o.shootFlavors)
	if err != nil {
		return errors.Wrap(err, "unable to render testrun")
	}

	if o.dryRun {
		fmt.Print(util.PrettyPrintStruct(runs))
		return nil
	}

	logger.Log.V(3).Info("starting watcher")

	watcher, err := watch.NewFromFile(logger.Log.WithName("watch"), o.tmKubeconfigPath, &o.watchOptions)
	if err != nil {
		return errors.Wrap(err, "unable to start testrun watch controller")
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

	collector, err := result.New(logger.Log.WithName("collector"), o.collectConfig, o.tmKubeconfigPath)
	if err != nil {
		return errors.Wrap(err, "unable to initialize collector")
	}
	if err := collector.PreRunShoots(o.shootParameters.GardenKubeconfigPath, runs); err != nil {
		return errors.Wrap(err, "unable to setup collector")
	}

	if err := testrunner.ExecuteTestruns(logger.Log.WithName("Execute"), &o.testrunnerConfig, runs, o.testrunNamePrefix, collector.RunExecCh); err != nil {
		return errors.Wrap(err, "unable to run testruns")
	}

	failed, err := collector.Collect(ctx, logger.Log.WithName("Collect"), o.testrunnerConfig.Watch.Client(), o.testrunnerConfig.Namespace, runs)
	if err != nil {
		return errors.Wrap(err, "unable to collect test output")
	}

	result.GenerateNotificationConfigForAlerting(runs.GetTestruns(), o.collectConfig.ConcourseOnErrorDir)

	logger.Log.Info("Testrunner finished")

	// Fail when one testrun is failed and we should fail on failed testruns.
	// Otherwise only fail when the testrun execution is erroneous.
	if runs.HasErrors() {
		return errors.New("At least one testrun failed. Stopping.")
	}
	// when there are one or many testruns in phase != succeeded
	if o.failOnError && len(failed) != 0 {
		msg := fmt.Sprintf("Something went wrong during testrun execution. Failed testruns: %s ", strings.Join(failed, ", "))
		return errors.New(msg)
	}

	return nil
}
