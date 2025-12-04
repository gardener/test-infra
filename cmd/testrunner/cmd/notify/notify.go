// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package notifycmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/slack"
)

var SucessSymbols = map[bool]string{
	true:  "✅",
	false: "❌",
}

const NA = "N/A"

var (
	overviewFileName        string
	githubUser              string
	githubPassword          string
	githubRepositoryName    string
	githubRepositoryVersion string

	slackChannel string
	slackToken   string
	cicdJobURL   string
)

// AddCommand adds notify to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(notifyCmd)
}

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Posts a result table of a previous run as table to slack.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		defer ctx.Done()

		component, err := result.EnhanceComponent(&componentdescriptor.Component{
			Name:    githubRepositoryName,
			Version: githubRepositoryVersion,
		}, githubUser, githubPassword)
		if err != nil {
			logger.Log.Error(err, "unable to get repository")
			os.Exit(1)
		}

		overview, err := result.DownloadAssetOverview(logger.Log, component, overviewFileName)
		if err != nil {
			logger.Log.Error(err, "unable to download asset from repository")
			os.Exit(1)
		}
		if len(overview.AssetOverviewItems) == 0 {
			logger.Log.Error(nil, "overview asset does not exist")
			os.Exit(1)
		}

		tableItems := parseOverviewToTableItems(overview)
		table, err := util.RenderTableForSlack(logger.Log, tableItems)
		if err != nil {
			return err
		}
		if table == "" {
			logger.Log.Info("no table to render")
			return nil
		}

		slackClient, err := slack.New(logger.Log, slackToken)
		if err != nil {
			return err
		}

		cicdJobURLFooter := ""
		if cicdJobURL != "" {
			cicdJobURLFooter = fmt.Sprintf("\nCI/CD Job: %s", cicdJobURL)
		}

		chunks := util.SplitString(fmt.Sprintf("%s\n%s", table, legend()), slack.MaxMessageLimit-100) // -100 to have space for header and footer messages
		if len(chunks) == 1 {
			return slackClient.PostMessage(slackChannel, fmt.Sprintf("%s\n```%s\n%s```%s", header(), table, legend(), cicdJobURLFooter))
		}

		for i, chunk := range chunks {
			message := fmt.Sprintf("```%s```", chunk)
			if i == 0 {
				message = fmt.Sprintf("%s\n%s", header(), message)
			}
			if i == len(chunks)-1 {
				message = fmt.Sprintf("%s%s", message, cicdJobURLFooter)
			}
			if err := slackClient.PostMessage(slackChannel, message); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	notifyCmd.Flags().StringVar(&githubRepositoryName, "github-repo", "", "Specify the Github repository that should be used to get the test results")
	notifyCmd.Flags().StringVar(&githubRepositoryVersion, "github-repo-version", "", "Specify the version fot the Github repository that should be used to get the test results")
	notifyCmd.Flags().StringVar(&githubUser, "github-user", os.Getenv("GITHUB_USER"), "On error dir which is used by Concourse.")
	notifyCmd.Flags().StringVar(&githubPassword, "github-password", os.Getenv("GITHUB_PASSWORD"), "Github password.")
	notifyCmd.Flags().StringVar(&overviewFileName, "overview", "", "Name of the overview asset file in the release.")
	notifyCmd.Flags().StringVar(&slackToken, "slack-token", "", "Client token to authenticate")
	notifyCmd.Flags().StringVar(&slackChannel, "slack-channel", "", "Client channel id to send the message to.")
	notifyCmd.Flags().StringVar(&cicdJobURL, "cicd-job-url", "", "CI/CD Job URL.")
	notifyCmd.Flags().StringVar(&cicdJobURL, "concourse-url", "", "Concourse job URL.")
	_ = notifyCmd.Flags().MarkDeprecated("concourse-url", "use --cicd-job-url instead")
}

func header() string {
	return "Integration Test results:"
}

func legend() string {
	return fmt.Sprintf(`
%s: Tests succeeded | %s: Tests failed | %s: Tests not applicable
`, SucessSymbols[true], SucessSymbols[false], NA)
}

func parseOverviewToTableItems(overview result.AssetOverview) (tableItems util.TableItems) {
	for _, overviewItem := range overview.AssetOverviewItems {
		meta := overviewItem.Dimension
		if meta.Cloudprovider == "" {
			// skip gardener tests
			continue
		} else {
			status := util.StatusSymbolFailure
			if overviewItem.Successful {
				status = util.StatusSymbolSuccess
			}
			item := &util.TableItem{
				Meta:         util.ItemMeta{CloudProvider: meta.Cloudprovider, TestrunID: overviewItem.Name, OperatingSystem: meta.OperatingSystem, KubernetesVersion: meta.KubernetesVersion, FlavorDescription: meta.Description},
				StatusSymbol: status,
			}
			tableItems = append(tableItems, item)
		}
	}
	return tableItems
}
