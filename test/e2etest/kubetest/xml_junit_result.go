package kubetest

import (
	"encoding/xml"
	"github.com/gardener/test-infra/test/e2etest/config"
	"regexp"
	"strings"
)

func (result *JunitXMLResult) CalculateAdditionalFields() {
	result.SuccessfulTests = result.ExecutedTests - result.FailedTests
	result.DurationInt = int(result.DurationFloat)
	regexpSigGroup := regexp.MustCompile(`^\[.*?]`)
	for i, _ := range result.Testcases {
		result.Testcases[i].calculateAdditionalFields(regexpSigGroup)
		// TODO: why is here result.Testcases[i].calculateAdditionalFields(regexpSigGroup) not possible, why is assigning to variable is required?
	}
}

func (testcase *TestcaseResult) calculateAdditionalFields(regexpSigGroup *regexp.Regexp) {
	testcase.SigGroup = regexpSigGroup.FindString(testcase.Name)
	if testcase.SkippedRaw != nil {
		testcase.Skipped = true
	}
	if testcase.FailureText == "" {
		testcase.Status = Success
		testcase.Successful = true
	} else {
		testcase.Status = Failure
		testcase.Successful = false
	}
	testcase.DurationInt = int(testcase.DurationFloat)
	testcase.TestDesc = config.DescriptionFile
	testcase.ExecutionGroup = strings.Join(config.TestcaseGroup, ",")
}

type JunitXMLResult struct {
	XMLName         xml.Name         `xml:"testsuite"`
	ExecutedTests   int              `xml:"tests,attr"`
	FailedTests     int              `xml:"failures,attr"`
	DurationFloat   float32          `xml:"time,attr"`
	DurationInt     int              `xml:"-"`
	Testcases       []TestcaseResult `xml:"testcase"`
	SuccessfulTests int              `xml:"-"`
}

type TestcaseResult struct {
	XMLName        xml.Name  `xml:"testcase" json:"-"`
	Name           string    `xml:"name,attr" json:"name"`
	Status         string    `xml:"-" json:"status"`
	SkippedRaw     *struct{} `xml:"skipped" json:"-"`
	Skipped        bool      `xml:"-" json:"-"`
	FailureText    string    `xml:"failure,omitempty" json:"failure.text,omitempty"`
	SystemOutput   string    `xml:"system-out,omitempty" json:"system-out,omitempty"`
	DurationFloat  float32   `xml:"time,attr" json:"-"`
	DurationInt    int       `xml:"-" json:"duration"`
	SigGroup       string    `xml:"-" json:"sig"`
	TestDesc       string    `xml:"-" json:"test_desc_file"`
	ExecutionGroup string    `xml:"-" json:"execution_group"`
	Successful     bool      `xml:"-" json:"successful"`
}
