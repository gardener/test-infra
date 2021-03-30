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

package result

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	trerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/slack"
)

func (c *Collector) postTestrunsSummaryInSlack(config Config, log logr.Logger, runs testrunner.RunList) error {
	if !config.PostSummaryInSlack {
		return nil
	}

	tableItems := parseTestrunsToTableItems(runs)
	table, err := util.RenderTableForSlack(log, tableItems)
	if err != nil {
		return errors.Wrap(err, "failed creating a table to post")
	}
	if table == "" {
		log.Info("no table to render")
		return nil
	}

	slackClient, err := slack.New(log, config.SlackToken)
	if err != nil {
		return errors.Wrap(err, "was not able to create slack client")
	}

	tmDashboardURL := ""
	if len(runs.GetTestruns()) > 0 {
		tmDashboardURL = runs.GetTestruns()[0].Annotations[common.AnnotationTMDashboardURL]
	}

	executionGroup := ""
	if len(runs.GetTestruns()) > 0 {
		executionGroup = runs.GetTestruns()[0].Labels[common.LabelTestrunExecutionGroup]
	}
	urlFooter := buildURLFooter(config.ConcourseURL, tmDashboardURL, config.GrafanaURL, executionGroup)

	chunks := util.SplitString(fmt.Sprintf("%s\n%s", table, legend()), slack.MaxMessageLimit-100) // -100 to have space for header and footer messages
	if len(chunks) == 1 {
		return slackClient.PostMessage(config.SlackChannel, fmt.Sprintf("%s\n```%s\n%s```\n%s", header(), table, legend(), urlFooter))
	}

	for i, chunk := range chunks {
		message := fmt.Sprintf("```%s```", chunk)
		if i == 0 {
			message = fmt.Sprintf("%s\n%s", header(), message)
		}
		if i == len(chunks)-1 {
			message = fmt.Sprintf("%s\n%s", message, urlFooter)
		}
		if err := slackClient.PostMessage(config.SlackChannel, message); err != nil {
			return errors.Wrap(err, "failed to post the slack message of test summary")
		}
	}

	return nil
}

func header() string {
	return "Integration Test results:"
}

func legend() string {
	return fmt.Sprintf(`
%s: Tests succeeded | %s: Tests failed | %s: Test execution error | %s: Tests not applicable
`, util.StatusSymbolSuccess, util.StatusSymbolFailure, util.StatusSymbolError, util.StatusSymbolNA)
}

func buildURLFooter(ccURL, tmDashboardURL, grafanaURL, executionGroup string) string {
	ccURLFooter := ""
	if ccURL != "" {
		ccURLFooter = fmt.Sprintf("<%s|Concourse Job>", ccURL)
	}

	tmDashboardURLFooter := ""
	if tmDashboardURL != "" {
		tmDashboardURLFooter = fmt.Sprintf("<%s|TM Native>", tmDashboardURL)
	}

	grafanaURLFooter := ""
	if grafanaURL != "" && executionGroup != "" {
		now := time.Now()
		from := now.Add(-12*time.Hour).Unix() * 1000
		to := now.Add(2*time.Hour).Unix() * 1000
		grafanaURL := fmt.Sprintf("%s&from=%d&to=%d&var-Filters=tm.tr.executionGroup.keyword%%7C=%%7C%s", grafanaURL, from, to, executionGroup)
		grafanaURLFooter = fmt.Sprintf("<%s|TM Grafana>", grafanaURL)
	}

	var parts []string
	if ccURLFooter != "" {
		parts = append(parts, ccURLFooter)
	}
	if tmDashboardURLFooter != "" {
		parts = append(parts, tmDashboardURLFooter)
	}
	if grafanaURLFooter != "" {
		parts = append(parts, grafanaURLFooter)
	}
	return strings.Join(parts, " • ")
}

func parseTestrunsToTableItems(runs testrunner.RunList) (tableItems util.TableItems) {
	for _, run := range runs {
		meta := run.Metadata
		if meta.CloudProvider == "" {
			// skip gardener tests
			continue
		} else {

			var status util.StatusSymbol
			if run.Error != nil && !trerrors.IsTimeout(run.Error) {
				status = util.StatusSymbolError
			} else {
				if run.Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess {
					status = util.StatusSymbolSuccess
				} else {
					status = util.StatusSymbolFailure
				}
			}

			var additionalDimensionInfo string
			if meta.AllowPrivilegedContainers != nil && !*meta.AllowPrivilegedContainers {
				additionalDimensionInfo = "NoPrivCtrs"
			}
			item := &util.TableItem{
				Meta:         util.ItemMeta{CloudProvider: meta.CloudProvider, TestrunID: meta.Testrun.ID, OperatingSystem: meta.OperatingSystem, KubernetesVersion: meta.KubernetesVersion, FlavorDescription: meta.FlavorDescription, AdditionalDimensionInfo: additionalDimensionInfo},
				StatusSymbol: status,
			}
			tableItems = append(tableItems, item)
		}
	}
	return tableItems
}
