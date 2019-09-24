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
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/analyse"
	telemetryctrl "github.com/gardener/test-infra/pkg/testrunner/telemetry"

	"github.com/spf13/cobra"
)

var (
	kubeconfigPath          string
	componentDescriptorPath string

	initialTimeoutString   string
	reconcileTimeoutString string

	resultDir string
)

// AddCommand adds run-testrun to a command.
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

		initialTimeout, err := time.ParseDuration(initialTimeoutString)
		if err != nil {
			logger.Log.Error(err, "unable to parse initial timeout duration")
			os.Exit(1)
		}
		reconcileTimeout, err := time.ParseDuration(reconcileTimeoutString)
		if err != nil {
			logger.Log.Error(err, "unable to parse reconcile timeout duration")
			os.Exit(1)
		}

		k8sClient, err := kubernetes.NewClientFromFile("", kubeconfigPath, kubernetes.WithClientOptions(client.Options{
			Scheme: kubernetes.GardenScheme,
		}))

		unhealthyShoots, err := GetUnhealthyShoots(logger.Log, ctx, k8sClient)
		if err != nil {
			logger.Log.Error(err, "unable to fetch unhealthy shoots")
			os.Exit(1)
		}

		telemetry, err := telemetryctrl.New(logger.Log.WithName("telemetry-controller"), 1*time.Second)
		if err != nil {
			logger.Log.Error(err, "unable to initialize telemetry controller")
			os.Exit(1)
		}
		if _, err := telemetry.Start(kubeconfigPath, resultDir); err != nil {
			logger.Log.Error(err, "unable to start telemetry controller")
			os.Exit(1)
		}

		// wait for update to finish
		logger.Log.Info("Wait for initial timeout..")
		time.Sleep(initialTimeout)

		// parse component descriptor and get latest gardener version
		cd, err := componentdescriptor.GetComponentsFromFile(componentDescriptorPath)
		if err != nil {
			logger.Log.Error(err, "unable to read component descriptor")
			os.Exit(1)
		}
		gardenerComponent := cd.Get("github.com/gardener/gardener")
		if gardenerComponent == nil {
			logger.Log.Error(nil, "gardener component not defined")
			os.Exit(1)
		}

		if err := WaitForGardenerUpdate(logger.Log.WithName("reconcile"), ctx, k8sClient, gardenerComponent.Version, unhealthyShoots, reconcileTimeout); err != nil {
			logger.Log.Error(err, "error waiting for all shoots to reconcile")
		}

		if err := telemetry.Stop(); err != nil {
			logger.Log.Error(err, "unable to stop telemetry controller and analyze metrics")
			os.Exit(1)
		}

		if err := analyse.Analyse(telemetry.RawResultsPath, "", common.ReportOutputFormatText); err != nil {
			logger.Log.Error(err, "unable to analyze measurement")
			os.Exit(1)
		}

		logger.Log.Info("finished collecting metrics")
	},
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

}
