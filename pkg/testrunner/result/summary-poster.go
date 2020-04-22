// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/slack"
	"github.com/go-logr/logr"
)

func (c *Collector) postTestrunsSummaryInSlack(config Config, log logr.Logger, runs testrunner.RunList) {
	if !config.PostSummaryInSlack {
		return
	}

	tableItems := parseTestrunsToTableItems(runs)
	table, err := util.RenderTableForSlack(log, tableItems)
	if err != nil {
		log.Error(err, "failed creating a table to post")
	}
	if table == "" {
		log.Info("no table to render")
		return
	}

	slackClient, err := slack.New(log, config.SlackToken)
	if err != nil {
		log.Error(err, "was not able to create slack client")
	}

	concourseURLFooter := ""
	if config.ConcourseURL != "" {
		concourseURLFooter = fmt.Sprintf("\nConcourse Job: %s", config.ConcourseURL)
	}

	if err := slackClient.PostMessage(config.SlackChannel, fmt.Sprintf("```%s\n%s\n%s```%s", header(), table, legend(), concourseURLFooter)); err != nil {
		log.Error(err, "failed to post the slack message of test summary")
	}
}

func header() string {
	return "Integration Test results:"
}

func legend() string {
	return fmt.Sprintf(`
%s: Tests succeeded | %s: Tests failed | %s: Tests not applicable
`, util.SucessSymbols[true], util.SucessSymbols[false], util.NA)
}

func parseTestrunsToTableItems(runs testrunner.RunList) (tableItems util.TableItems) {
	for _, run := range runs {
		meta := run.Metadata
		if meta.CloudProvider == "" {
			// skip gardener tests
			continue
		} else {
			item := &util.TableItem{
				Meta:    util.ItemMeta{CloudProvider: meta.CloudProvider, TestrunID: meta.Testrun.ID, OperatingSystem: meta.OperatingSystem, KubernetesVersion: meta.KubernetesVersion, FlavorDescription: meta.FlavorDescription},
				Success: run.Testrun.Status.Phase == argov1.NodeSucceeded,
			}
			tableItems = append(tableItems, item)
		}
	}
	return tableItems
}
