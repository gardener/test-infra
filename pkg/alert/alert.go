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
	"sort"
	"strings"
	"time"
)

type Alert struct {
	log                          logr.Logger
	es                           ElasticsearchConfig
	ctx                          context.Context
	minResultsPerTest            int
	evalTimeDays                 int
	successRateThreshhold        int
	continuousFailureThereshhold int
}

type TestDetails struct {
	Name                string
	Context             string
	FailedContinuously  bool
	LastFailedTimestamp string
	Cloudprovider       string
	OperatingSystem     string
	Landscape           string
	K8sVersion          string
	SuccessRate         float64
}

func (test TestDetails) String() string {
	return fmt.Sprintf("%s_%s_v%s_%s", test.Landscape, test.Cloudprovider, test.K8sVersion, test.Name)
}

type ElasticsearchConfig struct {
	Endpoint      string
	Authorization string
}

func New(log logr.Logger, es ElasticsearchConfig, ctx context.Context, minResultsPerTest int, evalTimeDays int, successRateThreshhold int, continuousFailureThereshhold int) (*Alert, error) {
	return &Alert{
		log:                          log,
		es:                           es,
		ctx:                          ctx,
		minResultsPerTest:            minResultsPerTest,
		evalTimeDays:                 evalTimeDays,
		successRateThreshhold:        successRateThreshhold,
		continuousFailureThereshhold: continuousFailureThereshhold,
	}, nil
}

type User struct{}

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

func (alert *Alert) FindFailedTests() map[string]TestDetails {
	distinctTestContexts := alert.getDistinctTestContexts()
	contextToTestDetailMap := extractTestDetailItems(distinctTestContexts)
	failedTests := removeSuccessfulTests(contextToTestDetailMap, alert.successRateThreshhold)
	alert.removeAlreadyFiledAlerts(failedTests)
	return failedTests
}

type alertDocs struct {
	Hits struct{
		AlertItems []struct{
			Source struct{
				TestName          string `json:"testContext"`
			}`json:"_source"`
		}`json:"hits"`
	}`json:"hits"`
}

func (alert *Alert) removeAlreadyFiledAlerts(tests map[string]TestDetails) {
	deleteOutdatedAlertsFromDB(alert)
	var alreadyFiledAlerts alertDocs
	alert.elasticRequest("/tm-alert*/_search", "GET", `{"size": 10000}`, &alreadyFiledAlerts)
	for _, alertItem := range alreadyFiledAlerts.Hits.AlertItems {
		delete(tests, alertItem.Source.TestName)
	}
}

func deleteOutdatedAlertsFromDB(alert *Alert) {
	deleteOutdatedAlertDocsPayload := fmt.Sprintf(`{
			"query": {
				"range": {
					"datetime": {
						"lt": "now-%dd"
					}
				}
			}
		}`, alert.evalTimeDays)
	alert.elasticRequest("/tm-alert*/_delete_by_query", "POST", deleteOutdatedAlertDocsPayload, nil)
}

func removeSuccessfulTests(contextToTestDetailMap map[string]TestDetails, successRateThreshhold int) map[string]TestDetails {
	for key, value := range contextToTestDetailMap {
		if !value.FailedContinuously && value.SuccessRate >= float64(successRateThreshhold) {
			delete(contextToTestDetailMap, key)
		}
	}
	return contextToTestDetailMap
}

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

func (alert *Alert) getDistinctTestContexts() testContextAggregation {
	payloadFormated := fmt.Sprintf(`{
		"size": 0,
		"query": {
			"bool": {
				"must": [
					{
						"match": {
							"type": "teststep"
						}
					},
					{
						"range": {
							"startTime": {
								"gte": "now-%dd"
							}
						}
					}
				],
				"should": [
					{
						"term": {
							"phase.keyword": "Failed"
						}
					},
					{
						"term": {
							"phase.keyword": "Succeeded"
						}
					},
					{
						"term": {
							"phase.keyword": "Skipped"
						}
					}
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
					"success_rate": {
						"avg": {
							"field": "pre.phaseNum"
						}
					},
					"success_trend": {
						"top_hits": {
							"sort": [
								{
									"startTime": {
										"order": "desc"
									}
								}
							],
							"_source": {
								"includes": [
									"pre.phaseNum"
								]
							},
							"size": %d
						}
					},
					"details": {
						"top_hits": {
							"sort": [
								{
									"startTime": {
										"order": "desc"
									}
								}
							],
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
	}`, alert.evalTimeDays, alert.continuousFailureThereshhold, alert.continuousFailureThereshhold)
	var testContextAggregation testContextAggregation
	alert.elasticGetSearchRequest(payloadFormated, &testContextAggregation)

	return testContextAggregation
}

func (alert *Alert) PostAlertToSlack(client slack.Client, channel string, failedTests map[string]TestDetails) error {
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
	alert.filePostedAlerts(failedTests)
	return nil
}

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
		if test.SuccessRate < float64(alert.successRateThreshhold) {
			failureReason = fmt.Sprintf("success rate < %d%%", alert.successRateThreshhold)
		} else if test.FailedContinuously {
			failureReason = fmt.Sprintf(">%d failures in row", alert.continuousFailureThereshhold-1)
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

func splitSlackMessage(message string, characterLimit int) []string {
	var messageSplits []string
	lines := strings.Split(message, "\n")
	messageSplitItem := ""
	for _, line := range lines {
		if len(messageSplitItem) + len(line) < characterLimit {
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

func (alert *Alert) elasticGetSearchRequest(payloadFormated string, result interface{}) {
	url := fmt.Sprintf("%s/testmachinery-%%2A/_search", alert.es.Endpoint)
	payload := strings.NewReader(payloadFormated)
	req, err := http.NewRequest("GET", url, payload)
	if err != nil {
		alert.log.Error(err, fmt.Sprintf("failed to create a http request for %s", url))
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Authorization", alert.es.Authorization)

	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		alert.log.Error(err, "failed to read response body %s")
	}
	if err := json.Unmarshal(body, result); err != nil {
		alert.log.Error(err, fmt.Sprintf("failed to unmarshal %s", string(body)))
	}
}

func (alert *Alert) elasticRequest(urlAttributes, httpMethod, payloadFormated string, result interface{}) {
	url := fmt.Sprintf("%s%s", alert.es.Endpoint, urlAttributes)
	payload := strings.NewReader(payloadFormated)
	req, err := http.NewRequest(strings.ToUpper(httpMethod), url, payload)
	if err != nil {
		alert.log.Error(err, fmt.Sprintf("failed to create a http request for %s", url))
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("Authorization", alert.es.Authorization)

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

func (alert *Alert) filePostedAlerts(tests map[string]TestDetails) {
	payload := generatePostedAlertsPayload(tests)
	alert.elasticRequest("/tm-alert/_doc/_bulk", "POST", payload, nil)
	alert.log.Info(fmt.Sprintf("filed %d tests as alerted in elasticsearch", len(tests)))
}

func generatePostedAlertsPayload(tests map[string]TestDetails) string {
	datetime := time.Now().UTC().Format("2006-01-02T15:04:05") + "Z"
	payload := ""
	for _, test := range tests {
		payload += fmt.Sprintf(`{ "index":{} }`+ "\n" + `{ "testContext":"%s","datetime":"%s" }`+ "\n", test.Context, datetime)
	}
	return payload
}
