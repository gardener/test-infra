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

package gardener_telemetry_cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/analyse"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	telemetryctrl "github.com/gardener/test-infra/pkg/testrunner/telemetry"
	"github.com/gardener/test-infra/pkg/util/gardener"
	kutil "github.com/gardener/test-infra/pkg/util/kubernetes"
)

var (
	kubeconfigPath          string
	componentDescriptorPath string
	ociOpts                 = &ociopts.Options{}

	initialTimeoutString   string
	reconcileTimeoutString string

	resultDir string
)

// AddCommand adds gardener-telemetry to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(gardenerTelemtryCmd)
}

var gardenerTelemtryCmd = &cobra.Command{
	Use:   "gardener-telemetry",
	Short: "Collects metrics during gardener updates until gardener is updated and all shoots are successfully reconciled",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer ctx.Done()
		logger.Log.Info("Start collecting metrics")
		if err := run(ctx); err != nil {
			panic(err)
		}
	},
}

func run(ctx context.Context) error {
	initialTimeout, err := time.ParseDuration(initialTimeoutString)
	if err != nil {
		return fmt.Errorf("unable to parse initial timeout duration: %w", err)
	}
	reconcileTimeout, err := time.ParseDuration(reconcileTimeoutString)
	if err != nil {
		return fmt.Errorf("unable to parse reconcile timeout duration: %w", err)
	}

	k8sClient, err := kutil.NewClientFromFile(kubeconfigPath, client.Options{
		Scheme: gardener.GardenScheme,
	})
	if err != nil {
		return fmt.Errorf("unable to create kubernetes client: %w", err)
	}

	unhealthyShoots, err := GetUnhealthyShoots(logger.Log, ctx, k8sClient)
	if err != nil {
		return fmt.Errorf("unable to fetch unhealthy shoots: %w", err)
	}

	telemetry, err := telemetryctrl.New(logger.Log.WithName("telemetry-controller"), 1*time.Second)
	if err != nil {
		return fmt.Errorf("unable to initialize telemetry controller: %w", err)
	}
	if err := telemetry.Start(kubeconfigPath, resultDir); err != nil {
		return fmt.Errorf("unable to start telemetry controller: %w", err)
	}

	// wait for update to finish
	logger.Log.Info("Wait for initial timeout..")
	time.Sleep(initialTimeout)

	// parse component descriptor and get latest gardener version
	cd, err := componentdescriptor.GetComponentsFromFileWithOCIOptions(ctx, logger.Log, ociOpts, componentDescriptorPath)
	if err != nil {
		return fmt.Errorf("unable to read component descriptor: %w", err)
	}
	gardenerComponent := cd.Get("github.com/gardener/gardener")
	if gardenerComponent == nil {
		return errors.New("gardener component not defined")
	}

	if err := WaitForGardenerUpdate(logger.Log.WithName("reconcile"), ctx, k8sClient, gardenerComponent.Version, unhealthyShoots, reconcileTimeout); err != nil {
		logger.Log.Error(err, "error waiting for all shoots to reconcile")
	}

	if err := telemetry.Stop(); err != nil {
		return fmt.Errorf("unable to stop telemetry controller and analyze metrics: %w", err)
	}

	if _, err := analyse.AnalyseDir(resultDir, "", common.ReportOutputFormatText); err != nil {
		return fmt.Errorf("unable to analyze measurement: %w", err)
	}

	logger.Log.Info("finished collecting metrics")
	return nil
}

func init() {
	// configuration flags
	gardenerTelemtryCmd.Flags().StringVar(&kubeconfigPath, "kubeconfig", os.Getenv("KUBECONFIG"), "Path to the gardener kubeconfig")
	if err := gardenerTelemtryCmd.MarkFlagFilename("kubeconfig"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "kubeconfig")
	}
	gardenerTelemtryCmd.Flags().StringVar(&componentDescriptorPath, "component-descriptor", "", "Path to component descriptor")
	if err := gardenerTelemtryCmd.MarkFlagFilename("component-descriptor"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "component-descriptor")
	}
	gardenerTelemtryCmd.Flags().StringVar(&resultDir, "result-dir", "/tmp/res", "Path to write the metricss")
	if err := gardenerTelemtryCmd.MarkFlagFilename("result-dir"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "result-dir")
	}

	gardenerTelemtryCmd.Flags().StringVar(&initialTimeoutString, "initial-timeout", "1m", "Initial timeout to wait for the update to start. Valid time units are 'ns', 'us' (or 'µs'), 'ms', 's', 'm', 'h'.")
	gardenerTelemtryCmd.Flags().StringVar(&reconcileTimeoutString, "reconcile-timeout", "10m", "Timeout to wait for all shoots to tbe reconciled. Valid time units are 'ns', 'us' (or 'µs'), 'ms', 's', 'm', 'h'.")

	ociOpts.AddFlags(gardenerTelemtryCmd.Flags())
}
