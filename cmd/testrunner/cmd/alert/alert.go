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
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/gardener/test-infra/pkg/alert"
	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/util/elasticsearch"
	"github.com/gardener/test-infra/pkg/util/slack"
)

var (
	elasticsearchEndpoint      string
	elasticsearchUser          string
	elasticsearchPass          string
	slackToken                 string
	slackChannel               string
	continuousFailureThreshold int
	evalTimeDays               int
	minSuccessRate             int
	testsSkip                  []string
	testsFocus                 []string
)

// AddCommand adds alert to a command.
func AddCommand(cmd *cobra.Command) {
	cmd.AddCommand(alertCmd)
}

var alertCmd = &cobra.Command{
	Use:   "alert",
	Short: "Evaluates recently completed testruns and sends alerts for failed  testruns if conditions are met.",
	Run: func(cmd *cobra.Command, args []string) {

		logger.Log.Info("Start testmachinery alerting")

		if err := validate(); err != nil {
			logger.Log.Error(err, "alert arguments validation failed")
			os.Exit(1)
		}

		slackClient, err := slack.New(logger.Log, slackToken)
		if err != nil {
			logger.Log.Error(err, "Cannot create slack client")
			os.Exit(1)
		}

		esClient, err := elasticsearch.NewClient(config.ElasticSearch{
			Endpoint: elasticsearchEndpoint,
			Username: elasticsearchUser,
			Password: elasticsearchPass,
		})
		if err != nil {
			logger.Log.Error(err, "Cannot create elasticsearch client")
			os.Exit(1)
		}

		alertConfig := alert.Config{
			ContinuousFailureThreshold:  continuousFailureThreshold,
			ESClient:                    esClient,
			EvalTimeDays:                evalTimeDays,
			SuccessRateThresholdPercent: minSuccessRate,
			TestsSkip:                   testsSkip,
			TestsFocus:                  testsFocus,
		}
		alertClient := alert.New(logger.Log.WithName("alert"), alertConfig)
		newFailedTests, recoveredTests, err := alertClient.FindFailedAndRecoveredTests()
		if err != nil {
			logger.Log.Error(err, "failed to find test items for alert and recover messages. Cannot sent slack message.")
			os.Exit(1)
		}
		if err := alertClient.PostAlertMessageToSlack(slackClient, slackChannel, newFailedTests); err != nil {
			logger.Log.Error(err, "failed to post an alert message to slack")
			os.Exit(1)
		}
		if err := alertClient.PostRecoverMessageToSlack(slackClient, slackChannel, recoveredTests); err != nil {
			logger.Log.Error(err, "failed to post a recover message to slack")
			os.Exit(1)
		}

		logger.Log.Info("finished alerting")
		os.Exit(0)
	},
}

func validate() error {
	if elasticsearchEndpoint == "" {
		return errors.New("elasticsearch-endpoint argument is required but empty")
	}
	if elasticsearchUser == "" {
		return errors.New("elasticsearch-user argument is required but empty")
	}
	if elasticsearchPass == "" {
		return errors.New("elasticsearch-pass argument is required but empty")
	}
	if slackToken == "" {
		return errors.New("slack-token argument is required but empty")
	}
	if slackChannel == "" {
		return errors.New("slack-channel argument is required but empty")
	}
	if continuousFailureThreshold == 0 {
		return errors.New("min-continuous-failures=0 is not allowed")
	}
	if evalTimeDays <= 0 {
		return errors.New("eval-time-days <= 0 is not allowed")
	}
	if minSuccessRate < 0 || minSuccessRate > 100 {
		return errors.New("min-success-rate must have a value between 0 and 100")
	}
	return nil
}

func init() {
	// parameter flags
	alertCmd.Flags().StringVar(&elasticsearchEndpoint, "elasticsearch-endpoint", "", "Elasticsearch endpoint URL")
	alertCmd.Flags().StringVar(&elasticsearchUser, "elasticsearch-user", "", "Elasticsearch username")
	alertCmd.Flags().StringVar(&elasticsearchPass, "elasticsearch-pass", "", "Elasticsearch password")
	alertCmd.Flags().StringVar(&slackToken, "slack-token", "", "Client token to authenticate")
	alertCmd.Flags().StringVar(&slackChannel, "slack-channel", "", "Client channel id to send the message to.")
	alertCmd.Flags().IntVar(&continuousFailureThreshold, "min-continuous-failures", 3, "if test fails >=n times send alert")
	alertCmd.Flags().IntVar(&evalTimeDays, "eval-time-days", 3, "time period to evaluate")
	alertCmd.Flags().IntVar(&minSuccessRate, "min-success-rate", 50, "if test success rate % falls below threshold, then post an alert")
	alertCmd.Flags().StringArrayVar(&testsSkip, "skip", make([]string, 0), "regexp to filter context test names e.g. 'e2e-untracked.*aws'")
	alertCmd.Flags().StringArrayVar(&testsFocus, "focus", make([]string, 0), "regexp to keep context test names e.g. 'e2e-untracked.*aws. Is executed after skip filter.'")
}
