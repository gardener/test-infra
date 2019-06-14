package kubetest

import (
	"encoding/xml"
	"github.com/gardener/test-infra/test/e2etest/config"
	"regexp"
)

func (result JunitXMLResult) CalculateAdditionalFields() {
	result.SuccessfulTests = result.ExecutedTests - result.FailedTests
	result.DurationInt = int(result.DurationFloat)
	regexpSigGroup := regexp.MustCompile(`^\[.*?]`)
	for i, _ := range result.Testcases {
		result.Testcases[i] = result.Testcases[i].calculateAdditionalFields(regexpSigGroup)
		// TODO: why is here result.Testcases[i].calculateAdditionalFields(regexpSigGroup) not possible, why is assigning to variable is required?
	}
}

func (testcase TestcaseResult) calculateAdditionalFields(regexpSigGroup *regexp.Regexp) TestcaseResult {
	testcase.SigGroup = regexpSigGroup.FindString(testcase.Name)
	if testcase.SkippedRaw != nil {
		testcase.Skipped = true
	}
	if testcase.FailureText == "" {
		testcase.Status = SUCCESS
	} else {
		testcase.Status = FAILURE
	}
	testcase.DurationInt = int(testcase.DurationFloat)
	testcase.TestDesc = config.DescriptionFile
	return testcase
}

type JunitXMLResult struct {
	XMLName         xml.Name `xml:"testsuite"`
	ExecutedTests   int      `xml:"tests,attr"`
	FailedTests     int      `xml:"failures,attr"`
	DurationFloat   float32  `xml:"time,attr"`
	DurationInt     int
	Testcases       []TestcaseResult `xml:"testcase"`
	SuccessfulTests int
}

type TestcaseResult struct {
	XMLName       xml.Name  `xml:"testcase" json:"-"`
	Name          string    `xml:"name,attr" json:"name"`
	Status        string    `json:"status" json:"-"`
	SkippedRaw    *struct{} `xml:"skipped" json:"-"`
	Skipped       bool      `json:"-"`
	FailureText   string    `xml:"failure" json:"failure.text,omitempty"`
	SystemOutput  string    `xml:"system-out" json:"system-out,omitempty"`
	DurationFloat float32   `xml:"time,attr" json:"-"`
	DurationInt   int       `json:"duration"`
	SigGroup      string    `json:"sig"`
	TestDesc      string    `json:"test_desc_file"`
}
