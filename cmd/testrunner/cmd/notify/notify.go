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
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/testrunner/result"
	"github.com/gardener/test-infra/pkg/util/slack"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"os"
	"reflect"
	"strings"
)

var SucessSymbols = map[bool]string{
	true:  "✅",
	false: "❌",
}

var (
	overviewFileName        string
	githubUser              string
	githubPassword          string
	githubRepositoryName    string
	githubRepositoryVersion string

	slackChannel string
	slackToken   string
)

// AddCommand adds run-testrun to a command.
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

		table, err := renderTableFromAsset(overview)
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

		return slackClient.PostMessage(slackChannel, fmt.Sprintf("```\n%s\n```", table))
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

func renderTableFromAsset(overview result.AssetOverview) (string, error) {
	writer := &strings.Builder{}
	table := tablewriter.NewWriter(writer)
	headerKeys := make(map[string]int, 0) // maps the header values to their index
	header := []string{""}
	dimensionRow := make(map[string]int, 0) // maps the dimension to their row
	content := make([][]string, 0)

	for _, asset := range overview.AssetOverviewItems {
		d := asset.Dimension
		if reflect.DeepEqual(d, testrunner.Dimension{}) {
			continue
		}
		_, ok := headerKeys[d.Cloudprovider]
		if !ok {
			header = append(header, d.Cloudprovider)
			headerKeys[d.Cloudprovider] = len(header) - 1
		}
	}

	for _, asset := range overview.AssetOverviewItems {
		d := asset.Dimension
		if reflect.DeepEqual(d, testrunner.Dimension{}) {
			continue
		}
		index, ok := headerKeys[d.Cloudprovider]
		if !ok {
			continue
		}

		dimensionKey := fmt.Sprintf("%s %s", d.KubernetesVersion, d.OperatingSystem)
		if d.Description != "" {
			dimensionKey = fmt.Sprintf("%s (%s)", dimensionKey, d.Description)
		}

		dRow, ok := dimensionRow[dimensionKey]
		if !ok {
			row := make([]string, len(header))
			row[0] = dimensionKey
			content = append(content, row)
			dimensionRow[dimensionKey] = len(content) - 1
			dRow = len(content) - 1
		}
		content[dRow][index] = SucessSymbols[asset.Successful]
	}
	if len(content) == 0 {
		return "", nil
	}

	table.SetHeader(header)
	table.AppendBulk(content)
	table.Render()
	return writer.String(), nil
}
