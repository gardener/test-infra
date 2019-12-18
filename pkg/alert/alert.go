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
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/slack"
	"github.com/go-logr/logr"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
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

//TestDetails describes a test which is used for alert message
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
	Successful          bool //Successful is true if success rate doesn't go below threshold and isn't failed continuously
}

//ElasticsearchBulkString creates an elastic search bulk string for ingestion
func (test TestDetails) ElasticsearchBulkString(datetime string) string {
	return fmt.Sprintf(`{ "index":{} }`+"\n"+`{ "testContext":"%s","datetime":"%s" }`+"\n", test.Context, datetime)
}

type ElasticsearchConfig struct {
	Endpoint      *url.URL
	User          string
	Pass          string
}

//Config represents alerting configuration
type Config struct {
	EvalTimeDays                int //EvalTimeDays time range to consinder for evaluation in days (now - n days before)
	SuccessRateThresholdPercent int //SuccessRateThresholdPercent if test success rate falls below threshold post an alert
	ContinuousFailureThreshold  int //ContinuousFailureThreshold if test fails >=n times send alert
	Elasticsearch               ElasticsearchConfig
	TestsToExclude              []string
}

func New(log logr.Logger, cfg Config) *Alert {
	return &Alert{
		log: log,
		cfg: cfg,
	}
}

//FindFailedAndRecoveredTests finds distinct, failed tests, that has not yet been posted on slack in recent n days
func (alert *Alert) FindFailedAndRecoveredTests() (map[string]TestDetails, map[string]TestDetails, error) {
	testAggregationsRaw, err := alert.retrieveTestAggregations()
	if err != nil {
		return nil, nil, err
	}
	contextToTestDetailMap := alert.extractTestDetailItems(testAggregationsRaw)
	if err := alert.deleteOutdatedAlertsFromDB(); err != nil {
		return nil, nil, err
	}
	alreadyFiledAlerts, err := alert.getFiledAlerts()
	if err != nil {
		return nil, nil, err
	}
	recoveredTests := alert.extractRecoveredTests(contextToTestDetailMap, alreadyFiledAlerts)
	newFailedTests := alert.removeSuccessfulTests(contextToTestDetailMap)
	if err := alert.removeExcludedTests(newFailedTests); err != nil {
		return nil, nil, err
	}
	alert.removeAlreadyFiledAlerts(newFailedTests, alreadyFiledAlerts)
	return newFailedTests, recoveredTests, nil
}

func (alert *Alert) retrieveTestAggregations() (TestContextAggregation, error) {
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

	var testContextAggregation TestContextAggregation
	if err := alert.elasticRequest("/testmachinery-*/_search", http.MethodGet, payloadFormated, &testContextAggregation); err != nil {
		return TestContextAggregation{}, errors.Wrap(err, "failed to retrieve testmachinery test aggregations from elasticsearch")
	}
	alert.log.V(3).Info(fmt.Sprintf("retrieved %d distinct test aggregations", len(testContextAggregation.Aggs.TestContext.TestDetailsRaw)))

	return testContextAggregation, nil
}

//extractTestDetailItems parses raw elasticsearch test aggregations into test details
func (alert *Alert) extractTestDetailItems(testContextAggregation TestContextAggregation) map[string]TestDetails {
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
		successful := !testFailedContinuously && int(testDoc.SuccessRate.Value) >= alert.cfg.SuccessRateThresholdPercent
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
			Successful:          successful,
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
	alert.log.V(3).Info(fmt.Sprintf("removed %d/%d tests, because they are successful", testsSizeBefore-len(contextToTestDetailMap), testsSizeBefore))
	return contextToTestDetailMap
}

//removeAlreadyFiledAlerts removes tests based on given test exclude regexp patterns
func (alert *Alert) removeExcludedTests(tests map[string]TestDetails) error {
	testsSizeBefore := len(tests)
	for _, patternStr := range alert.cfg.TestsToExclude {
		alert.log.V(3).Info(fmt.Sprintf("filtering out tests with expression %s", patternStr))
		for testContext, _ := range tests {
			matched, err := regexp.MatchString(patternStr, testContext)
			if err != nil {
				return errors.Wrapf(err,"failed to apply regexep %s", patternStr)
			}
			if matched {
				delete(tests, testContext)
			}
		}
	}
	alert.log.V(3).Info(fmt.Sprintf("filtered %d/%d tests using exclusion patterns", testsSizeBefore-len(tests), testsSizeBefore))
	return nil
}

//removeAlreadyFiledAlerts filters out tests that have already been alerted in recent n days
func (alert *Alert) removeAlreadyFiledAlerts(tests map[string]TestDetails, alreadyFiledAlerts alertDocs) {
	testsSizeBefore := len(tests)
	for _, alertItem := range alreadyFiledAlerts.Hits.AlertItems {
		delete(tests, alertItem.Source.TestName)
	}
	alert.log.V(3).Info(fmt.Sprintf("%d/%d tests alerts have been discarded, since they have already been posted in slack", testsSizeBefore-len(tests), testsSizeBefore))
}

//getFiledAlerts gets list of existing alert docs in elasticsearch
func (alert *Alert) getFiledAlerts() (alertDocs, error) {
	var alreadyFiledAlerts alertDocs
	if err := alert.elasticRequest("/tm-alert*/_search", http.MethodGet, `{"size": 10000}`, &alreadyFiledAlerts); err != nil {
		return alertDocs{}, errors.Wrap(err, "failed to get elasticsearch alert items")
	}
	alert.log.V(3).Info(fmt.Sprintf("retrieved %d already posted test alerts", len(alreadyFiledAlerts.Hits.AlertItems)))
	return alreadyFiledAlerts, nil
}

//deleteOutdatedAlertDocsPayload deletes all elasticsearch documents that are older than n days
func (alert *Alert) deleteOutdatedAlertsFromDB() error {
	alert.log.V(3).Info("delete outdated elasticsearch alert docs")
	deleteOutdatedAlertDocsPayload := fmt.Sprintf(`{ "query": { "range": { "datetime": { "lt": "now-%dd" } } } }`, alert.cfg.EvalTimeDays)
	if err := alert.elasticRequest("/tm-alert*/_delete_by_query", http.MethodPost, deleteOutdatedAlertDocsPayload, nil); err != nil {
		return errors.Wrap(err, "failed to delete outdated elasticsearch alert items")
	}
	return nil
}

//PostAlertMessageToSlack posts alerts to slack
func (alert *Alert) PostAlertMessageToSlack(client slack.Client, channel string, failedTests map[string]TestDetails) error {
	if len(failedTests) == 0 {
		alert.log.Info("no new failed tests found, nothing to post in slack")
		return nil
	}
	message := createAlertMessage(failedTests, alert)
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
	alert.log.Info("Sent slack alert message", "failing tests", len(failedTests))
	if err := alert.filePostedAlerts(failedTests); err != nil {
		return err
	}
	return nil
}

//createAlertMessage creates the alert message
func createAlertMessage(failedTests map[string]TestDetails, alert *Alert) string {
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
			util.StringDefault(test.Cloudprovider, "-"),
			util.StringDefault(test.K8sVersion, "-"),
			util.StringDefault(test.OperatingSystem, "-"),
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

//PostRecoverMessageToSlack posts alerts to slack
func (alert *Alert) PostRecoverMessageToSlack(client slack.Client, channel string, recoveredTests map[string]TestDetails) error {
	if len(recoveredTests) == 0 {
		alert.log.Info("no new recovered tests found, nothing to post in slack")
		return nil
	}
	message := createRecoverMessage(recoveredTests)
	splittedMessage := splitSlackMessage(message, 3900)
	messagePrefix := "*🍏 Testmachinery Tests Got Healthy:* \n"
	for i, messageSplitItem := range splittedMessage {
		if i != 0 {
			messagePrefix = ""
		}
		if err := client.PostMessage(channel, fmt.Sprintf("%s```%s```", messagePrefix, messageSplitItem)); err != nil {
			return errors.Wrap(err, "failed to post a slack message")
		}
		time.Sleep(1200 * time.Millisecond) // need to wait 1 sec due to slack limits
	}
	alert.log.Info("Sent slack recover message", "recovered tests", len(recoveredTests))

	testNames := make([]string, 0, len(recoveredTests))
	for key := range recoveredTests {
		testNames = append(testNames, key)
	}
	if err := alert.deleteRecoveredTestsFromAlertIndex(testNames); err != nil {
		return err
	}
	return nil
}

//createAlertMessage creates the alert message
func createRecoverMessage(recoveredTests map[string]TestDetails) string {
	sortedKeys := make([]string, 0, len(recoveredTests))
	for k := range recoveredTests {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	writer := &strings.Builder{}
	table := tablewriter.NewWriter(writer)
	header := []string{"Test", "Landscape", "Provider", "K8s Ver", "OS"}

	content := make([][]string, 0)
	row := 0
	for _, mapKey := range sortedKeys {
		test := recoveredTests[mapKey]
		newRow := []string{test.Name,
			test.Landscape,
			util.StringDefault(test.Cloudprovider, "-"),
			util.StringDefault(test.K8sVersion, "-"),
			util.StringDefault(test.OperatingSystem, "-"),
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

func (alert *Alert) deleteRecoveredTestsFromAlertIndex(testNames []string) error {
	alert.log.V(3).Info("delete recovered tests from elasticsearch tm-alert index")
	deleteOutdatedAlertDocsPayload := fmt.Sprintf(`{ "query": { "terms" : { "testContext.keyword" : ["%s"] } } }`, strings.Join(testNames,`", "`))
	if err := alert.elasticRequest("/tm-alert*/_delete_by_query", http.MethodPost, deleteOutdatedAlertDocsPayload, nil); err != nil {
		return errors.Wrapf(err, "failed to delete recovered elasticsearch test docs")
	}
	return nil
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

//elasticRequest send HTTP request to elasticsearch
func (alert *Alert) elasticRequest(urlAttributes, httpMethod, payloadFormated string, result interface{}) error {
	alert.cfg.Elasticsearch.Endpoint.Path = path.Join(urlAttributes)
	requestUrl := alert.cfg.Elasticsearch.Endpoint.String()
	payload := strings.NewReader(payloadFormated)
	alert.log.V(3).Info(fmt.Sprintf("creating HTTP %s request on %s", httpMethod, requestUrl))
	req, err := http.NewRequest(httpMethod, requestUrl, payload)
	if err != nil {
		return errors.Wrapf(err, "failed to create a http request for %s", requestUrl)
	}

	req.SetBasicAuth(alert.cfg.Elasticsearch.User, alert.cfg.Elasticsearch.Pass)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute HTTP request")
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read response body")
	}
	if result != nil {
		if err := json.Unmarshal(body, result); err != nil {
			return errors.Wrapf(err, "failed to unmarshal %s", string(body))
		}
	}
	return nil
}

//filePostedAlerts posts test contexts to elasticsearch
func (alert *Alert) filePostedAlerts(tests map[string]TestDetails) error {
	payload := generatePostedAlertsPayload(tests)
	if err := alert.elasticRequest("/tm-alert/_doc/_bulk", "POST", payload, nil); err != nil {
		return errors.Wrap(err, "failed to store alerted tests in elasticsearch")
	}
	alert.log.V(3).Info(fmt.Sprintf("filed %d tests as alerted in elasticsearch", len(tests)))
	return nil
}

func (alert *Alert) extractRecoveredTests(testContextToTestMap map[string]TestDetails, filedAlerts alertDocs) map[string]TestDetails {
	recoveredTests := make(map[string]TestDetails)
	for _, filedAlert := range filedAlerts.Hits.AlertItems {
		test, ok := testContextToTestMap[filedAlert.Source.TestName]
		if ok && test.Successful {
			recoveredTests[filedAlert.Source.TestName] = test
		}
	}
	return recoveredTests
}

//generatePostedAlertsPayload generates a bulk payload of test context docs
func generatePostedAlertsPayload(tests map[string]TestDetails) string {
	datetime := time.Now().UTC().Format(time.RFC3339)
	payload := ""
	for _, test := range tests {
		payload += test.ElasticsearchBulkString(datetime)
	}
	return payload
}

// elasticsearch distinct names aggregation structure
type TestContextAggregation struct {
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
