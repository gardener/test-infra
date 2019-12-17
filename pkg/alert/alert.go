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
	"encoding/json"
	"fmt"
	"github.com/gardener/test-infra/pkg/util/slack"
	"github.com/go-logr/logr"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Alert struct {
	log logr.Logger
	cfg Config
	ctx context.Context
}

//TestDetails describes a test
type TestDetails struct {
	Name                string  //Name test name
	Context             string  //Context is the concatenation of name and several test dimensions
	FailedContinuously  bool    //FailedContinuously true if n recent test runs were failing in a row
	LastFailedTimestamp string  //LastFailedTimestamp timestamp of last failed test execution
	SuccessRate         float64 //SuccessRate of recent n days
	Cloudprovider       string
	OperatingSystem     string
	Landscape           string
	K8sVersion          string
}

type ElasticsearchConfig struct {
	Authorization string //Authorization basic auth token in format "Basic R2fyZGVUZXI6bWR7Y1IxWkgycGpsNTdNNG1DbnQ="
	Endpoint      string
}

//Config represents alerting configuration
type Config struct {
	EvalTimeDays                int //EvalTimeDays time range to consinder for evaluation in days (now - n days before)
	SuccessRateThresholdPercent int //SuccessRateThresholdPercent if test success rate falls below threshold post an alert
	ContinuousFailureThreshold  int //ContinuousFailureThreshold if test fails >=n times send alert
	Elasticsearch               ElasticsearchConfig
	Logger                      logr.Logger
	Context                     context.Context
	TestsToExclude              []string
}

func New(cfg Config) (*Alert, error) {
	return &Alert{
		log: cfg.Logger,
		cfg: cfg,
		ctx: cfg.Context,
	}, nil
}

//FindFailedTests finds distinct, failed tests, that has not yet been posted on slack in recent n days
func (alert *Alert) FindFailedTests() map[string]TestDetails {
	testAggregationsRaw := alert.retrieveTestAggregations()
	contextToTestDetailMap := extractTestDetailItems(testAggregationsRaw)
	failedTests := alert.removeSuccessfulTests(contextToTestDetailMap)
	alert.removeExcludedTests(failedTests)
	alert.removeAlreadyFiledAlerts(failedTests)
	return failedTests
}

func (alert *Alert) retrieveTestAggregations() testContextAggregation {
	payloadFormated := fmt.Sprintf(`{
		"size": 0,
		"query": {
			"bool": {
				"must": [
					{ "match": { "type": "teststep" } },
					{ "range": { "startTime": { "gte": "now-%dd" } } }
				],
				"should": [
					{ "term": { "phase.keyword": "Failed" } },
					{ "term": { "phase.keyword": "Succeeded" } },
					{ "term": { "phase.keyword": "Skipped" } }
				],
				"minimum_should_match": 1
			}
		},
		"aggs": {
			"test_context": {
				"terms": {
					"script": {
						"source": "def landscape = ''; def k8s_version = ''; def name = ''; def provider = 'none'; def os = ''; if (doc['tm.cloudprovider.keyword'].size() != 0) { provider = doc['tm.cloudprovider.keyword'].value;  } if (doc['tm.landscape.keyword'].size() != 0) { landscape = doc['tm.landscape.keyword'].value; } if (doc['tm.operating_system.keyword'].size() != 0) { os = doc['tm.operating_system.keyword'].value; } if (doc['tm.k8s_version.keyword'].size() != 0) { k8s_version = '_v' + doc['tm.k8s_version.keyword'].value; } if (doc['name.keyword'].size() != 0) { name = doc['name.keyword'].value; } name + '_' + landscape + '_' + provider + k8s_version + '_' + os;",
						"lang": "painless"
					},
					"min_doc_count": %d,
					"size": 10000
				},
				"aggs": {
					"success_rate": { "avg": { "field": "pre.phaseNum" } },
					"success_trend": {
						"top_hits": {
							"sort": [ { "startTime": { "order": "desc" } } ],
							"_source": { "includes": [ "pre.phaseNum" ] },
							"size": %d
						}
					},
					"details": {
						"top_hits": {
							"sort": [ { "startTime": { "order": "desc" } } ],
							"_source": {
								"includes": [
									"name",
									"startTime",
									"pre.clusterDomain",
									"tm.landscape",
									"tm.tr.id",
									"tm.cloudprovider",
									"tm.k8s_version",
									"tm.operating_system"
								]
							},
							"size": 1
						}
					}
				}
			}
		}
	}`, alert.cfg.EvalTimeDays, alert.cfg.ContinuousFailureThreshold, alert.cfg.ContinuousFailureThreshold)

	var testContextAggregation testContextAggregation
	alert.elasticRequest(`/testmachinery-*/_search`, "GET", payloadFormated, &testContextAggregation)
	alert.log.Info(fmt.Sprintf("retrieved %d distinct test aggregations", len(testContextAggregation.Aggs.TestContext.TestDetailsRaw)))

	return testContextAggregation
}

//extractTestDetailItems parses raw elasticsearch test aggregations into test details
func extractTestDetailItems(testContextAggregation testContextAggregation) map[string]TestDetails {
	contextToTestDetailMap := make(map[string]TestDetails)
	for _, testDoc := range testContextAggregation.Aggs.TestContext.TestDetailsRaw {
		testDocDetails := testDoc.Details.Hits.Hits[0].Source
		testFailedContinuously := true
		for _, trendItem := range testDoc.SuccessTrend.Hits.Hits {
			if trendItem.Source.Pre.PhaseNum != 0 {
				testFailedContinuously = false
				break
			}
		}
		parsedTestDetail := TestDetails{
			Name:                testDocDetails.Name,
			LastFailedTimestamp: testDocDetails.LastTimestamp,
			OperatingSystem:     testDocDetails.TM.OperatingSystem,
			K8sVersion:          testDocDetails.TM.K8sVersion,
			Cloudprovider:       testDocDetails.TM.Cloudprovider,
			Landscape:           testDocDetails.TM.Landscape,
			SuccessRate:         testDoc.SuccessRate.Value,
			Context:             testDoc.Testcontext,
			FailedContinuously:  testFailedContinuously,
		}
		contextToTestDetailMap[testDoc.Testcontext] = parsedTestDetail
	}
	return contextToTestDetailMap
}

//removeSuccessfulTests removes tests from input map, that are not exceeding given thresholds and are therefore considered successful
func (alert *Alert) removeSuccessfulTests(contextToTestDetailMap map[string]TestDetails) map[string]TestDetails {
	testsSizeBefore := len(contextToTestDetailMap)
	for key, value := range contextToTestDetailMap {
		if !value.FailedContinuously && value.SuccessRate >= float64(alert.cfg.SuccessRateThresholdPercent) {
			delete(contextToTestDetailMap, key)
		}
	}
	alert.log.Info(fmt.Sprintf("removed %d/%d tests, because they are successful", testsSizeBefore-len(contextToTestDetailMap), testsSizeBefore))
	return contextToTestDetailMap
}

//removeAlreadyFiledAlerts removes tests based on given test exclude regexp patterns
func (alert *Alert) removeExcludedTests(tests map[string]TestDetails) {
	testsSizeBefore := len(tests)
	for _, patternStr := range alert.cfg.TestsToExclude {
		alert.log.Info(fmt.Sprintf("filtering out tests with expression %s", patternStr))
		for testContext, _ := range tests {
			matched, err := regexp.MatchString(patternStr, testContext)
			if err != nil {
				alert.log.Error(err, fmt.Sprintf("failed to apply regexep %s. Error: %s", patternStr))
			}
			if matched {
				delete(tests, testContext)
			}
		}
	}
	alert.log.Info(fmt.Sprintf("filtered %d/%d tests using exclusion patterns", testsSizeBefore - len(tests), testsSizeBefore))
}

//removeAlreadyFiledAlerts filters out tests that have already been alerted in recent n days
func (alert *Alert) removeAlreadyFiledAlerts(tests map[string]TestDetails) {
	deleteOutdatedAlertsFromDB(alert)
	testsSizeBefore := len(tests)
	var alreadyFiledAlerts alertDocs
	alert.elasticRequest(`/tm-alert*/_search`, "GET", `{"size": 10000}`, &alreadyFiledAlerts)
	alert.log.Info(fmt.Sprintf("retrieved %d already posted test alerts", len(alreadyFiledAlerts.Hits.AlertItems)))
	for _, alertItem := range alreadyFiledAlerts.Hits.AlertItems {
		delete(tests, alertItem.Source.TestName)
	}
	alert.log.Info(fmt.Sprintf("%d/%d tests alerts have been discarded, since they have already been posted in slack", testsSizeBefore - len(tests), testsSizeBefore))
}

//deleteOutdatedAlertDocsPayload deletes all elasticsearch documents that are older than n days
func deleteOutdatedAlertsFromDB(alert *Alert) {
	alert.log.Info("delete outdated elasticsearch alert docs")
	deleteOutdatedAlertDocsPayload := fmt.Sprintf(`{ "query": { "range": { "datetime": { "lt": "now-%dd" } } } }`, alert.cfg.EvalTimeDays)
	alert.elasticRequest(`/tm-alert*/_delete_by_query`, "POST", deleteOutdatedAlertDocsPayload, nil)
}

//PostAlertToSlack posts alerts to slack
func (alert *Alert) PostAlertToSlack(client slack.Client, channel string, failedTests map[string]TestDetails) error {
	if len(failedTests) == 0 {
		alert.log.Info("no new failed tests found, nothing to post in slack")
		return nil
	}
	message := createMessage(failedTests, alert)
	splitedMessage := splitSlackMessage(message, 3900)
	messagePrefix := "*🔥 New Testmachinery Alerts:* \n"
	for i, messageSplitItem := range splitedMessage {
		if i != 0 {
			messagePrefix = ""
		}
		if err := client.PostMessage(channel, fmt.Sprintf("%s```%s```", messagePrefix, messageSplitItem)); err != nil {
			return err
		}
		time.Sleep(1200 * time.Millisecond) // need to wait 1 sec due to slack limits
	}
	alert.log.Info("Sent slack alert message of %d failing tests", len(failedTests))
	alert.filePostedAlerts(failedTests)
	return nil
}

//createMessage creates the alert message
func createMessage(failedTests map[string]TestDetails, alert *Alert) string {
	sortedKeys := make([]string, 0, len(failedTests))
	for k := range failedTests {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	writer := &strings.Builder{}
	table := tablewriter.NewWriter(writer)
	header := []string{"Test", "Landscape", "Provider", "K8s Ver", "OS", "Success", "Alert Reason", "Last failure"}

	content := make([][]string, 0)
	row := 0
	for _, mapKey := range sortedKeys {
		test := failedTests[mapKey]
		failureReason := ""
		if test.SuccessRate < float64(alert.cfg.SuccessRateThresholdPercent) {
			failureReason = fmt.Sprintf("success rate < %d%%", alert.cfg.SuccessRateThresholdPercent)
		} else if test.FailedContinuously {
			failureReason = fmt.Sprintf(">%d failures in row", alert.cfg.ContinuousFailureThreshold-1)
		}
		newRow := []string{test.Name,
			test.Landscape,
			stringOrDefault(test.Cloudprovider, "-"),
			stringOrDefault(test.K8sVersion, "-"),
			stringOrDefault(test.OperatingSystem, "-"),
			fmt.Sprintf("%d%%", int(test.SuccessRate)),
			failureReason,
			test.LastFailedTimestamp,
		}
		content = append(content, newRow)
		row++
	}

	table.SetHeader(header)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.AppendBulk(content)
	table.Render()
	return writer.String()
}

//splitSlackMessage split message line wise based on given characters limit
func splitSlackMessage(message string, charactersLimit int) []string {
	var messageSplits []string
	lines := strings.Split(message, "\n")
	messageSplitItem := ""
	for _, line := range lines {
		if len(messageSplitItem)+len(line) < charactersLimit {
			messageSplitItem += "\n" + line
		} else {
			messageSplits = append(messageSplits, messageSplitItem)
			messageSplitItem = line
		}
	}
	messageSplits = append(messageSplits, messageSplitItem)
	return messageSplits
}

func stringOrDefault(input, defaultValue string) string {
	if input == "" {
		return defaultValue
	}
	return input
}

//elasticRequest send HTTP request to elasticsearch
func (alert *Alert) elasticRequest(urlAttributes, httpMethod, payloadFormated string, result interface{}) {
	url := fmt.Sprintf("%s%s", alert.cfg.Elasticsearch.Endpoint, urlAttributes)
	payload := strings.NewReader(payloadFormated)
	alert.log.Info(fmt.Sprintf("creating HTTP %s request on %s", httpMethod, url))
	req, err := http.NewRequest(strings.ToUpper(httpMethod), url, payload)
	if err != nil {
		alert.log.Error(err, fmt.Sprintf("failed to create a http request for %s", url))
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Authorization", alert.cfg.Elasticsearch.Authorization)

	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		alert.log.Error(err, "failed to read response body %s")
	}
	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			alert.log.Error(err, fmt.Sprintf("failed to unmarshal %s", string(body)))
		}
	}
}

//filePostedAlerts posts test contexts to elasticsearch
func (alert *Alert) filePostedAlerts(tests map[string]TestDetails) {
	payload := generatePostedAlertsPayload(tests)
	alert.elasticRequest("/tm-alert/_doc/_bulk", "POST", payload, nil)
	alert.log.Info(fmt.Sprintf("filed %d tests as alerted in elasticsearch", len(tests)))
}

//generatePostedAlertsPayload generates a bulk payload of test context docs
func generatePostedAlertsPayload(tests map[string]TestDetails) string {
	datetime := time.Now().UTC().Format("2006-01-02T15:04:05") + "Z"
	payload := ""
	for _, test := range tests {
		payload += fmt.Sprintf(`{ "index":{} }`+"\n"+`{ "testContext":"%s","datetime":"%s" }`+"\n", test.Context, datetime)
	}
	return payload
}

// MessageRequest defines a default slack request for a message
type MessageRequest struct {
	Channel string `json:"channel"`
	Text    string `json:"text,omitempty"`
	AsUser  bool   `json:"as_user,omitempty"`
}

// Response defines a slack response
type Response struct {
	Ok      bool         `json:"ok"`
	Message *interface{} `json:"message"`
	Error   *string      `json:"error"`
}

// elasticsearch distinct names aggregation structure
type testContextAggregation struct {
	Aggs struct {
		TestContext struct {
			TestDetailsRaw []struct {
				Testcontext string `json:"key"`
				Details     struct {
					Hits struct {
						Hits []struct {
							Source struct {
								Name          string `json:"name"`
								LastTimestamp string `json:"startTime"`
								TM            struct {
									Cloudprovider   string `json:"cloudprovider"`
									K8sVersion      string `json:"k8s_version"`
									OperatingSystem string `json:"operating_system"`
									Landscape       string `json:"landscape"`
								} `json:"tm"`
							} `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"details"`
				SuccessTrend struct {
					Hits struct {
						Hits []struct {
							Source struct {
								Pre struct {
									PhaseNum int `json:"phaseNum"`
								} `json:"pre"`
							} `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"success_trend"`
				SuccessRate struct {
					Value float64 `json:"value"`
				} `json:"success_rate"`
			} `json:"buckets"`
		} `json:"test_context"`
	} `json:"aggregations"`
}

// elasticsearch alert docs structure
type alertDocs struct {
	Hits struct {
		AlertItems []struct {
			Source struct {
				TestName string `json:"testContext"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
