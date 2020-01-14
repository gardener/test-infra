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

//ElasticsearchConfig represents the elasticsearch configuration
type ElasticsearchConfig struct {
	Endpoint *url.URL
	User     string
	Pass     string
}

//Config represents alerting configuration
type Config struct {
	EvalTimeDays                int //EvalTimeDays time range to consinder for evaluation in days (now - n days before)
	SuccessRateThresholdPercent int //SuccessRateThresholdPercent if test success rate falls below threshold post an alert
	ContinuousFailureThreshold  int //ContinuousFailureThreshold if test fails >=n times send alert
	Elasticsearch               ElasticsearchConfig
	TestsToExclude              []string
}

//New creates a new instance of alert
func New(log logr.Logger, cfg Config) *Alert {
	return &Alert{
		log: log,
		cfg: cfg,
	}
}

//FindFailedAndRecoveredTests finds distinct, failed tests, that has not yet been posted on slack in recent n days
func (alerter *Alert) FindFailedAndRecoveredTests() (map[string]TestDetails, map[string]TestDetails, error) {
	testAggregationsRaw, err := alerter.retrieveTestAggregations()
	if err != nil {
		return nil, nil, err
	}
	contextToTestDetailMap := alerter.extractTestDetailItems(testAggregationsRaw)
	if err := alerter.deleteOutdatedAlertsFromDB(); err != nil {
		return nil, nil, err
	}
	alreadyFiledAlerts, err := alerter.getFiledAlerts()
	if err != nil {
		return nil, nil, err
	}
	recoveredTests := alerter.extractRecoveredTests(contextToTestDetailMap, alreadyFiledAlerts)
	newFailedTests := alerter.removeSuccessfulTests(contextToTestDetailMap)
	if err := alerter.removeExcludedTests(newFailedTests); err != nil {
		return nil, nil, err
	}
	alerter.removeAlreadyFiledAlerts(newFailedTests, alreadyFiledAlerts)
	return newFailedTests, recoveredTests, nil
}

//retrieveTestAggregations retrieves test aggregations from elasticsearch
func (alerter *Alert) retrieveTestAggregations() (TestContextAggregation, error) {
	payloadFormated := alerter.generateESAggregationPayload()
	var testContextAggregation TestContextAggregation
	if err := alerter.elasticRequest("/testmachinery-*/_search", http.MethodGet, payloadFormated, &testContextAggregation); err != nil {
		return TestContextAggregation{}, errors.Wrap(err, "failed to retrieve testmachinery test aggregations from elasticsearch")
	}
	alerter.log.V(3).Info(fmt.Sprintf("retrieved %d distinct test aggregations", len(testContextAggregation.Aggs.TestContext.TestDetailsRaw)))

	return testContextAggregation, nil
}

//extractTestDetailItems parses raw elasticsearch test aggregations into test details
func (alerter *Alert) extractTestDetailItems(testContextAggregation TestContextAggregation) map[string]TestDetails {
	contextToTestDetailMap := make(map[string]TestDetails)
	for _, testDoc := range testContextAggregation.Aggs.TestContext.TestDetailsRaw {
		testDocDetails := testDoc.Details.Hits.Docs[0].Source
		testFailedContinuously := true
		for _, trendItem := range testDoc.SuccessTrend.Hits.Hits {
			if trendItem.Source.Pre.PhaseNum != 0 {
				testFailedContinuously = false
				break
			}
		}
		successful := !testFailedContinuously && int(testDoc.SuccessRate.Value) >= alerter.cfg.SuccessRateThresholdPercent
		parsedTestDetail := TestDetails{
			Name:                testDocDetails.Name,
			LastFailedTimestamp: testDocDetails.LastTimestamp,
			OperatingSystem:     testDocDetails.TM.OperatingSystem,
			K8sVersion:          testDocDetails.TM.K8sVersion,
			Cloudprovider:       testDocDetails.TM.Cloudprovider,
			Landscape:           testDocDetails.TM.Landscape,
			TestrunID:           testDocDetails.TM.Testrun.ID,
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
func (alerter *Alert) removeSuccessfulTests(contextToTestDetailMap map[string]TestDetails) map[string]TestDetails {
	testsSizeBefore := len(contextToTestDetailMap)
	for key, value := range contextToTestDetailMap {
		if !value.FailedContinuously && value.SuccessRate >= float64(alerter.cfg.SuccessRateThresholdPercent) {
			delete(contextToTestDetailMap, key)
		}
	}
	alerter.log.V(3).Info(fmt.Sprintf("removed %d/%d tests, because they are successful", testsSizeBefore-len(contextToTestDetailMap), testsSizeBefore))
	return contextToTestDetailMap
}

//removeAlreadyFiledAlerts removes tests based on given test exclude regexp patterns
func (alerter *Alert) removeExcludedTests(tests map[string]TestDetails) error {
	testsSizeBefore := len(tests)
	for _, patternStr := range alerter.cfg.TestsToExclude {
		alerter.log.V(3).Info(fmt.Sprintf("filtering out tests with expression %s", patternStr))
		for testContext, _ := range tests {
			matched, err := regexp.MatchString(patternStr, testContext)
			if err != nil {
				return errors.Wrapf(err, "failed to apply regexep %s", patternStr)
			}
			if matched {
				delete(tests, testContext)
			}
		}
	}
	alerter.log.V(3).Info(fmt.Sprintf("filtered %d/%d tests using exclusion patterns", testsSizeBefore-len(tests), testsSizeBefore))
	return nil
}

//removeAlreadyFiledAlerts filters out tests that have already been alerted in recent n days
func (alerter *Alert) removeAlreadyFiledAlerts(tests map[string]TestDetails, alreadyFiledAlerts AlertDocs) {
	testsSizeBefore := len(tests)
	for _, alertItem := range alreadyFiledAlerts.Hits.AlertItems {
		delete(tests, alertItem.Source.Context)
	}
	alerter.log.V(3).Info(fmt.Sprintf("%d/%d tests alerts have been discarded, since they have already been posted in slack", testsSizeBefore-len(tests), testsSizeBefore))
}

//getFiledAlerts gets list of existing alert docs in elasticsearch
func (alerter *Alert) getFiledAlerts() (AlertDocs, error) {
	var alreadyFiledAlerts AlertDocs
	if err := alerter.elasticRequest("/tm-alerter*/_search", http.MethodGet, `{"size": 10000}`, &alreadyFiledAlerts); err != nil {
		return AlertDocs{}, errors.Wrap(err, "failed to get elasticsearch alerter items")
	}
	alerter.log.V(3).Info(fmt.Sprintf("retrieved %d already posted test alerts", len(alreadyFiledAlerts.Hits.AlertItems)))
	return alreadyFiledAlerts, nil
}

//deleteOutdatedAlertDocsPayload deletes all elasticsearch documents that are older than n days
func (alerter *Alert) deleteOutdatedAlertsFromDB() error {
	alerter.log.V(3).Info("delete outdated elasticsearch alerter docs")
	deleteOutdatedAlertDocsPayload := fmt.Sprintf(`{ "query": { "range": { "datetime": { "lt": "now-%dd" } } } }`, alerter.cfg.EvalTimeDays)
	if err := alerter.elasticRequest("/tm-alerter*/_delete_by_query", http.MethodPost, deleteOutdatedAlertDocsPayload, nil); err != nil {
		return errors.Wrap(err, "failed to delete outdated elasticsearch alerter items")
	}
	return nil
}

//PostAlertMessageToSlack posts alerts to slack
func (alerter *Alert) PostAlertMessageToSlack(client slack.Client, channel string, failedTests map[string]TestDetails) error {
	if len(failedTests) == 0 {
		alerter.log.Info("no new failed tests found, nothing to post in slack")
		return nil
	}
	message := createAlertMessage(failedTests, alerter)
	splitedMessage := splitSlackMessage(message, 3900)
	messagePrefix := "*🔥 New Testmachinery Alerts:* \n"
	messageSuffix := ""
	for i, messageSplitItem := range splitedMessage {
		if i != 0 {
			messagePrefix = ""
		}
		if i == (len(splitedMessage) - 1) {
			messageSuffix = "\nCheckout list of all active alerts and links to logs at https://kibana.ingress.cicdes.core.shoot.live.k8s-hana.ondemand.com/app/kibana#/discover/2eb705d0-339d-11ea-96c2-197bcb58cf5f?_g=(filters%3A!()%2CrefreshInterval%3A(pause%3A!t%2Cvalue%3A0)%2Ctime%3A(from%3Anow-7d%2Cto%3Anow))"
		}
		if err := client.PostMessage(channel, fmt.Sprintf("%s```%s```%s", messagePrefix, messageSplitItem, messageSuffix)); err != nil {
			return err
		}
		time.Sleep(1200 * time.Millisecond) // need to wait 1 sec due to slack limits
	}
	alerter.log.Info("Sent slack alerter message", "failing tests", len(failedTests))
	if err := alerter.filePostedAlerts(failedTests); err != nil {
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
func (alerter *Alert) PostRecoverMessageToSlack(client slack.Client, channel string, recoveredTests map[string]TestDetails) error {
	if len(recoveredTests) == 0 {
		alerter.log.Info("no new recovered tests found, nothing to post in slack")
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
	alerter.log.Info("Sent slack recover message", "recovered tests", len(recoveredTests))

	testNames := make([]string, 0, len(recoveredTests))
	for key := range recoveredTests {
		testNames = append(testNames, key)
	}
	if err := alerter.deleteRecoveredTestsFromAlertIndex(testNames); err != nil {
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

func (alerter *Alert) deleteRecoveredTestsFromAlertIndex(testNames []string) error {
	alerter.log.V(3).Info("delete recovered tests from elasticsearch tm-alerter index")
	deleteOutdatedAlertDocsPayload := fmt.Sprintf(`{ "query": { "terms" : { "testContext.keyword" : ["%s"] } } }`, strings.Join(testNames, `", "`))
	if err := alerter.elasticRequest("/tm-alerter*/_delete_by_query", http.MethodPost, deleteOutdatedAlertDocsPayload, nil); err != nil {
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
func (alerter *Alert) elasticRequest(urlAttributes, httpMethod, payloadFormated string, result interface{}) error {
	alerter.cfg.Elasticsearch.Endpoint.Path = path.Join(urlAttributes)
	requestUrl := alerter.cfg.Elasticsearch.Endpoint.String()
	payload := strings.NewReader(payloadFormated)
	alerter.log.V(3).Info(fmt.Sprintf("creating HTTP %s request on %s", httpMethod, requestUrl))
	req, err := http.NewRequest(httpMethod, requestUrl, payload)
	if err != nil {
		return errors.Wrapf(err, "failed to create a http request for %s", requestUrl)
	}

	req.SetBasicAuth(alerter.cfg.Elasticsearch.User, alerter.cfg.Elasticsearch.Pass)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cache-Control", "no-cache")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to execute HTTP request")
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		errorBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read failed HTTP response body")
		}
		return errors.Wrapf(err, "failed to execute HTTP request: HTTP Status Code %d Content: %s", res.StatusCode, string(errorBody))
	}

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
func (alerter *Alert) filePostedAlerts(tests map[string]TestDetails) error {
	payload := alerter.generatePostedAlertsPayload(tests)
	if err := alerter.elasticRequest("/tm-alerter/_doc/_bulk", http.MethodPost, payload, nil); err != nil {
		return errors.Wrap(err, "failed to store alerted tests in elasticsearch")
	}
	alerter.log.V(3).Info(fmt.Sprintf("filed %d tests as alerted in elasticsearch", len(tests)))
	return nil
}

func (alerter *Alert) extractRecoveredTests(testContextToTestMap map[string]TestDetails, filedAlerts AlertDocs) map[string]TestDetails {
	recoveredTests := make(map[string]TestDetails)
	for _, filedAlert := range filedAlerts.Hits.AlertItems {
		test, ok := testContextToTestMap[filedAlert.Source.Context]
		if ok && test.Successful {
			recoveredTests[filedAlert.Source.Context] = test
		}
	}
	return recoveredTests
}

//generatePostedAlertsPayload generates a bulk payload of test context docs
func (alerter *Alert) generatePostedAlertsPayload(tests map[string]TestDetails) string {
	datetime := time.Now().UTC().Format(time.RFC3339)
	payload := ""
	for _, test := range tests {
		test.FiledAlertDataTime = datetime
		bulkString, err := test.ElasticsearchBulkString()
		if err != nil {
			alerter.log.Error(err, "Failed to marshal test details item", "item", test)
		}
		payload += bulkString
	}
	return payload
}

//generateESAggregationPayload format elasticsearch aggregation payload
func (alerter *Alert) generateESAggregationPayload() string {
	return fmt.Sprintf(`{
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
	}`, alerter.cfg.EvalTimeDays, alerter.cfg.ContinuousFailureThreshold, alerter.cfg.ContinuousFailureThreshold)
}
