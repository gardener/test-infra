package alert

import (
	"encoding/json"
	"fmt"
)

// TestContextAggregation elasticsearch distinct names aggregation structure
type TestContextAggregation struct {
	Aggs struct {
		TestContext struct {
			TestDetailsRaw []struct {
				Testcontext string `json:"key"`
				Details     struct {
					Hits struct {
						Docs []ESTestmachineryDoc `json:"hits"`
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

type ESTestmachineryDoc struct {
	Source struct {
		Name          string `json:"name"`
		LastTimestamp string `json:"startTime"`
		TM            struct {
			Cloudprovider   string `json:"cloudprovider"`
			K8sVersion      string `json:"k8s_version"`
			OperatingSystem string `json:"operating_system"`
			Landscape       string `json:"landscape"`
			Testrun         struct {
				ID string `json:"id"`
			} `json:"tr"`
		} `json:"tm"`
	} `json:"_source"`
}

// AlertDocs elasticsearch alert docs structure
type AlertDocs struct {
	Hits struct {
		AlertItems []struct {
			Source TestDetails `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

//TestDetails describes a test which is used for alert message
type TestDetails struct {
	Name                string  `json:"name"`                         //Name test name
	Context             string  `json:"testContext"`                  //Context is the concatenation of name and several test dimensions
	FailedContinuously  bool    `json:"failedContinuously"` //FailedContinuously true if n recent test runs were failing in a row
	LastFailedTimestamp string  `json:"lastTimeFailed"`               //LastFailedTimestamp timestamp of last failed test execution
	SuccessRate         float64 `json:"successRate"`                  //SuccessRate of recent n days
	FiledAlertDataTime  string  `json:"datetime"`                     //FiledAlertDataTime is the timestamp when the alert has been filed in slack
	Successful          bool    `json:"-"`                            //Successful is true if success rate doesn't go below threshold and isn't failed continuously
	Cloudprovider       string  `json:"cloudprovider,omitempty"`
	OperatingSystem     string  `json:"operatingSystem,omitempty"`
	Landscape           string  `json:"landscape"`
	K8sVersion          string  `json:"k8sVersion,omitempty"`
	TestrunID           string  `json:"testrunID"`
}

//ElasticsearchBulkString creates an elastic search bulk string for ingestion
func (test TestDetails) ElasticsearchBulkString() (string, error) {
	testDetailsMarshaled, err := json.Marshal(test)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`{ "index":{} }`+"\n%s\n", string(testDetailsMarshaled)), nil
}
