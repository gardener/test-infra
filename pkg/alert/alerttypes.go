package alert

//TestContextAggregation elasticsearch distinct names aggregation structure
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
		} `json:"tm"`
	} `json:"_source"`
}

//AlertDocs elasticsearch alert docs structure
type AlertDocs struct {
	Hits struct {
		AlertItems []struct {
			Source struct {
				TestName string `json:"testContext"`
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}
