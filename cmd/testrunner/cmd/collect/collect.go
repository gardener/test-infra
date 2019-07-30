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

package collectcmd

import (
	"context"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	"os"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var (
	tmKubeconfigPath string
	namespace        string

	testrunName string

	outputDirPath           string
	elasticSearchConfigName string
	s3Endpoint              string
	s3SSL                   bool
	concourseOnErrorDir     string
)

// AddCommand adds run-testrun to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(collectCmd)
}

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collects results from a completed testrun.",
	Run: func(cmd *cobra.Command, args []string) {
		if debug, _ := cmd.Flags().GetBool("debug"); debug {
			log.SetLevel(log.DebugLevel)
			log.Warn("Set debug log level")

			cmd.DebugFlags()
		}
		ctx := context.Background()
		defer ctx.Done()
		log.Info("Start testmachinery testrunner")

		collectConfig := &result.Config{
			OutputDir:           outputDirPath,
			ESConfigName:        elasticSearchConfigName,
			S3Endpoint:          s3Endpoint,
			S3SSL:               s3SSL,
			ConcourseOnErrorDir: concourseOnErrorDir,
		}
		log.Debugln(util.PrettyPrintStruct(collectConfig))

		tmClient, err := kubernetes.NewClientFromFile("", tmKubeconfigPath, client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		})
		if err != nil {
			log.Fatalf("Cannot build kubernetes client from %s: %s", tmKubeconfigPath, err.Error())
		}

		tr := &tmv1beta1.Testrun{}
		err = tmClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: testrunName}, tr)
		if err != nil {
			log.Fatalf("Cannot fetch testrun %s from cluster", testrunName)
		}

		run := &testrunner.Run{
			Testrun:  tr,
			Metadata: testrunner.MetadataFromTestrun(tr),
		}

		_, err = result.Collect(collectConfig, tmClient, namespace, []*testrunner.Run{run})
		if err != nil {
			log.Fatal(err.Error())
		}

		log.Info("Finished collecting testrun results.")
	},
}

func init() {
	// configuration flags
	collectCmd.Flags().StringVar(&tmKubeconfigPath, "tm-kubeconfig-path", os.Getenv("KUBECONFIG"), "Path to the testmachinery cluster kubeconfig")
	if err := collectCmd.MarkFlagFilename("tm-kubeconfig-path"); err != nil {
		log.Debug(err)
	}
	collectCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace where the testrun should be deployed.")

	collectCmd.Flags().StringVarP(&testrunName, "tr-name", "t", "", "Name of the testrun to collect results.")
	if err := collectCmd.MarkFlagRequired("tr-name"); err != nil {
		log.Debug(err)
	}

	// parameter flags
	collectCmd.Flags().StringVarP(&outputDirPath, "output-dir-path", "o", "./testout", "The filepath where the summary should be written to.")
	collectCmd.Flags().StringVar(&elasticSearchConfigName, "es-config-name", "", "The elasticsearch secret-server config name.")
	collectCmd.Flags().StringVar(&s3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	collectCmd.Flags().BoolVar(&s3SSL, "s3-ssl", false, "S3 has SSL enabled.")
	collectCmd.Flags().StringVar(&concourseOnErrorDir, "concourse-onError-dir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")

}
