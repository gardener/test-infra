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

package alert

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/gardener/test-infra/pkg/alert"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/util/slack"
	"github.com/spf13/cobra"
)

var (
	elasticsearchAPI string
	elasticsearchUser string
	elasticsearchPass string
	slackToken        string
	slackChannel        string
)

// AddCommand adds alert to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(alertCmd)
}

var alertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Evaluates recently completed testruns and sends alerts for failed  testruns if conditions are met.",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		defer ctx.Done()

		logger.Log.Info("Start testmachinery alerting")

		slackClient, err := slack.New(logger.Log, slackToken)
		if err != nil {
			logger.Log.Error(err, "Cannot create slack client")
			return
		}

		basicAuthToken := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", elasticsearchUser, elasticsearchPass))))
		alertClient, _ := alert.New(logger.Log.WithName("alert"), alert.ElasticsearchConfig{Endpoint: elasticsearchAPI, Authorization: basicAuthToken}, ctx, 3, 7, 50, 3)
		failedTests := alertClient.FindFailedTests()
		if len(failedTests) > 0 {
			if err := alertClient.PostAlertToSlack(slackClient, slackChannel, failedTests); err != nil {
				logger.Log.Error(err, "failed to post an alert to slack")
			}
		}

		logger.Log.Info("finished alerting")
		return
	},
}

func init() {
	// parameter flags
	alertCmd.Flags().StringVar(&elasticsearchAPI, "elasticsearch-endpoint", "", "Elasticsearch endpoint URL")
	alertCmd.Flags().StringVar(&elasticsearchUser, "elasticsearch-user", "", "Elasticsearch username")
	alertCmd.Flags().StringVar(&elasticsearchPass, "elasticsearch-pass", "", "Elasticsearch password")
	alertCmd.Flags().StringVar(&slackToken, "slack-token", "", "Client token to authenticate")
	alertCmd.Flags().StringVar(&slackChannel, "slack-channel", "", "Client channel id to send the message to.")
}
