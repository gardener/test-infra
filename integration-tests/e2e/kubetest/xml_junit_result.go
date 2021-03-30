package kubetest

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	"github.com/gardener/test-infra/integration-tests/e2e/config"
)

func (result *JunitXMLResult) CalculateAdditionalFields() {
	result.SuccessfulTests = result.ExecutedTests - result.FailedTests
	result.DurationInt = int(result.DurationFloat)
	regexpSigGroup := regexp.MustCompile(`^\[.*?]`)
	for i, _ := range result.Testcases {
		result.Testcases[i].calculateAdditionalFields(regexpSigGroup)
		// TODO: why is here result.ExplicitTestcases[i].calculateAdditionalFields(regexpSigGroup) not possible, why is assigning to variable is required?
	}
}

func (testcase *TestcaseResult) calculateAdditionalFields(regexpSigGroup *regexp.Regexp) {
	testcase.SigGroup = regexpSigGroup.FindString(testcase.Name)

	testcase.ContextedName = fmt.Sprintf("%s_v%s_%s", config.CloudProvider, config.K8sReleaseMajorMinor, testcase.Name)
	if testcase.SkippedRaw != nil {
		testcase.Skipped = true
	}
	if testcase.FailureText == "" {
		testcase.Status = Success
		testcase.Successful = true
		testcase.StatusShort = "S"
		testcase.SuccessRate = 100
	} else {
		testcase.Status = Failure
		testcase.Successful = false
		testcase.StatusShort = "F"
		testcase.SuccessRate = 0
	}
	testcase.DurationInt = int(testcase.DurationFloat)
	testcase.TestDesc = config.DescriptionFile
	testcase.ExecutionGroup = strings.Join(config.TestcaseGroup, ",")
	testcase.K8sMajor = config.K8sReleaseMajorMinor
}

type JunitXMLResult struct {
	XMLName         xml.Name         `xml:"testsuite"`
	ExecutedTests   int              `xml:"tests,attr"`
	FailedTests     int              `xml:"failures,attr"`
	Errors          int              `xml:"errors,attr"`
	DurationFloat   float32          `xml:"time,attr"`
	Testcases       []TestcaseResult `xml:"testcase"`
	DurationInt     int              `xml:"-"` // calculated
	SuccessfulTests int              `xml:"-"` // calculated
}

type TestcaseResult struct {
	XMLName        xml.Name  `xml:"testcase" json:"-"`
	Name           string    `xml:"name,attr" json:"name"`
	Status         string    `xml:"-" json:"status"` // calculated
	SkippedRaw     *struct{} `xml:"skipped" json:"-"`
	Skipped        bool      `xml:"-" json:"-"` // calculated
	FailureText    string    `xml:"failure,omitempty" json:"failure.text,omitempty"`
	SystemOutput   string    `xml:"system-out,omitempty" json:"system-out,omitempty"`
	DurationFloat  float32   `xml:"time,attr" json:"-"`
	DurationInt    int       `xml:"-" json:"duration"`        // calculated
	SigGroup       string    `xml:"-" json:"sig"`             // calculated
	TestDesc       string    `xml:"-" json:"test_desc_file"`  // calculated
	ExecutionGroup string    `xml:"-" json:"execution_group"` // calculated
	Successful     bool      `xml:"-" json:"successful"`      // calculated
	Flaked         int       `xml:"-" json:"flaked"`          // calculated
	StatusShort    string    `xml:"-" json:"status_short"`    // calculated
	ContextedName  string    `xml:"-" json:"contexted_name"`  // calculated
	SuccessRate    int       `xml:"-" json:"success_rate"`    // calculated
	K8sMajor       string    `xml:"-" json:"k8s_major"`       // calculated

}
