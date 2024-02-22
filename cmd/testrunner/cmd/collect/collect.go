// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package collectcmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	"github.com/gardener/test-infra/pkg/util"
	kutil "github.com/gardener/test-infra/pkg/util/kubernetes"
)

var (
	tmKubeconfigPath string
	namespace        string

	testrunName string
)

var collectConfig = result.Config{}

// AddCommand adds collect to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(collectCmd)
}

var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "Collects results from a completed testrun.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer ctx.Done()
		logger.Log.Info("Start testmachinery testrunner")

		logger.Log.V(3).Info(util.PrettyPrintStruct(collectConfig))

		tmClient, err := kutil.NewClientFromFile(tmKubeconfigPath, client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		})
		if err != nil {
			logger.Log.Error(err, fmt.Sprintf("Cannot build kubernetes client from %s", tmKubeconfigPath))
			os.Exit(1)
		}

		tr := &tmv1beta1.Testrun{}
		err = tmClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: testrunName}, tr)
		if err != nil {
			logger.Log.Error(err, "unable to fetch testrun %s from cluster", "testrun", testrunName)
			os.Exit(1)
		}

		run := &testrunner.Run{
			Testrun:  tr,
			Metadata: metadata.FromTestrun(tr),
		}

		collector, err := result.New(logger.Log.WithName("collector"), collectConfig, tmKubeconfigPath)
		if err != nil {
			logger.Log.Error(err, "unable to initialize collector")
			os.Exit(1)
		}
		_, err = collector.Collect(ctx, logger.Log.WithName("Collect"), tmClient, namespace, []*testrunner.Run{run})
		if err != nil {
			logger.Log.Error(err, "unable to collect result", "testrun", testrunName)
			os.Exit(1)
		}

		logger.Log.Info("finished collecting testrun results.")
	},
}

func init() {
	// configuration flags
	collectCmd.Flags().StringVar(&tmKubeconfigPath, "tm-kubeconfig-path", os.Getenv("KUBECONFIG"), "Path to the testmachinery cluster kubeconfig")
	if err := collectCmd.MarkFlagFilename("tm-kubeconfig-path"); err != nil {
		logger.Log.Error(err, "mark flag filename", "flag", "tm-kubeconfig-path")
	}
	collectCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace where the testrun should be deployed.")

	collectCmd.Flags().StringVarP(&testrunName, "tr-name", "t", "", "Name of the testrun to collect results.")
	if err := collectCmd.MarkFlagRequired("tr-name"); err != nil {
		logger.Log.Error(err, "mark flag required", "flag", "tr-name")
	}
	collectCmd.Flags().StringVar(&collectConfig.ComponentDescriptorPath, "component-descriptor-path", "", "Path to the component descriptor (BOM) of the current landscape.")

	// parameter flags
	collectCmd.Flags().StringVar(&collectConfig.ConcourseOnErrorDir, "concourse-onError-dir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")

	// asset upload
	collectCmd.Flags().BoolVar(&collectConfig.UploadStatusAsset, "upload-status-asset", false, "Upload testrun status as a github release asset.")
	collectCmd.Flags().StringVar(&collectConfig.GithubUser, "github-user", os.Getenv("GITHUB_USER"), "On error dir which is used by Concourse.")
	collectCmd.Flags().StringVar(&collectConfig.GithubPassword, "github-password", os.Getenv("GITHUB_PASSWORD"), "Github password.")
	collectCmd.Flags().StringArrayVar(&collectConfig.AssetComponents, "asset-component", []string{}, "The github components to which the testrun status shall be attached as an asset.")
	collectCmd.Flags().StringVar(&collectConfig.AssetPrefix, "asset-prefix", "", "Prefix of the asset name.")

	// slack notification
	collectCmd.Flags().StringVar(&collectConfig.SlackToken, "slack-token", "", "Client token to authenticate")
	collectCmd.Flags().StringVar(&collectConfig.SlackChannel, "slack-channel", "", "Client channel id to send the message to.")
	collectCmd.Flags().StringVar(&collectConfig.ConcourseURL, "concourse-url", "", "Concourse job URL.")
	collectCmd.Flags().BoolVar(&collectConfig.PostSummaryInSlack, "post-summary-in-slack", false, "Post testruns summary in slack.")

	// DEPRECATED FLAGS
	collectCmd.Flags().StringP("output-dir-path", "o", "./testout", "The filepath where the summary should be written to.")
	collectCmd.Flags().String("es-config-name", "sap_internal", "DEPRECATED: The elasticsearch secret-server config name.")
	collectCmd.Flags().String("es-endpoint", "", "endpoint of the elasticsearch instance")
	collectCmd.Flags().String("es-username", "", "username to authenticate against a elasticsearch instance")
	collectCmd.Flags().String("es-password", "", "password to authenticate against a elasticsearch instance")
	collectCmd.Flags().String("s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	collectCmd.Flags().Bool("s3-ssl", false, "S3 has SSL enabled.")
	_ = collectCmd.Flags().MarkDeprecated("output-dir-path", "DEPRECATED: will not we used anymore")
	_ = collectCmd.Flags().MarkDeprecated("es-config-name", "DEPRECATED: will not we used anymore")
	_ = collectCmd.Flags().MarkDeprecated("es-endpoint", "DEPRECATED: will not we used anymore")
	_ = collectCmd.Flags().MarkDeprecated("es-username", "DEPRECATED: will not we used anymore")
	_ = collectCmd.Flags().MarkDeprecated("es-password", "DEPRECATED: will not we used anymore")
	_ = collectCmd.Flags().MarkDeprecated("s3-endpoint", "DEPRECATED: will not we used anymore")
	_ = collectCmd.Flags().MarkDeprecated("s3-ssl", "DEPRECATED: will not we used anymore")
}
