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

package notifycmd

import (
	"context"
	"fmt"
	"os"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	"github.com/gardener/test-infra/pkg/util/slack"
	"github.com/spf13/cobra"
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
			logger.Log.Error(nil, "overview sset does not exist")
			os.Exit(1)
		}

		table, err := renderTableFromAsset(logger.Log, overview)
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

		return slackClient.PostMessage(slackChannel, fmt.Sprintf("```%s\n%s\n%s```", header(), table, legend()))
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
}

func header() string {
	return "Integration Test Results:"
}

func legend() string {
	return fmt.Sprintf(`
%s: Tests succeeded | %s: Tests failed | %s: Tests not applicable
`, SucessSymbols[true], SucessSymbols[false], NA)
}
