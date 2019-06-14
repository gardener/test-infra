package kubetest

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gardener/test-infra/test/e2etest/config"
	"github.com/gardener/test-infra/test/e2etest/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"
)

const (
	E2eLogFileNamePattern   = "e2e.log$"
	JunitXmlFileNamePattern = `junit_\d+.xml$`
	TestSummaryFileName     = "test_summary.json"
)

// Analyze analyzes junit.xml files and e2e.log files, which are dumped by kubetest and provides a resulting test suite summary and results for each testcase individually. These results are then written to the export dir as files.
func Analyze(kubetestResultsPath string) Summary {
	e2eLogFilePaths := util.GetFilesByPattern(kubetestResultsPath, E2eLogFileNamePattern)
	summary := analyzeE2eLogs(e2eLogFilePaths)
	junitXMLFilePaths := util.GetFilesByPattern(kubetestResultsPath, JunitXmlFileNamePattern)
	analyzeJunitXMLs(junitXMLFilePaths)
	return summary
}

func analyzeJunitXMLs(junitXMLFilePaths []string) {
	for _, junitXMLPath := range junitXMLFilePaths {
		file, err := os.Open(junitXMLPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		junitXml, err := UnmarshalJunitXMLResult(file.Name())
		if err != nil {
			log.Fatal(errors.Wrapf(err, "Couldn't unmarshal %s", file.Name()))
		}
		for _, testcase := range junitXml.Testcases {
			if testcase.Skipped {
				continue
			}
			testcaseJSON, err := json.MarshalIndent(testcase, "", " ")
			if err != nil {
				log.Fatal(errors.Wrapf(err, "Couldn't marshal testsuite summary %s", testcaseJSON))
			}

			jsonFileName := fmt.Sprintf("test-%s.json", strconv.FormatInt(time.Now().UnixNano(), 10))
			testcaseJsonFilePath := path.Join(config.ExportPath, jsonFileName)
			if err := ioutil.WriteFile(testcaseJsonFilePath, testcaseJSON, 0644); err != nil {
				log.Fatal(errors.Wrapf(err, "Couldn't write %s to file", testcaseJsonFilePath))
			}
		}
	}
}

func analyzeE2eLogs(e2eLogFilePaths []string) Summary {
	summary := Summary{TestsuiteSuccessful: false, FailedTestcases: 0, SuccessfulTestcases: 0, ExecutedTestcases: 0, TestsuiteDuration: 0, FlakedTestcases: 0, DescriptionFile: config.DescriptionFile, Flaked: false}
	regexpRanSpecs := regexp.MustCompile(`Ran (?P<TestcasesRan>\d+).*Specs.in (?P<TestSuiteDuration>\d+)`)
	regexpPassedFailed := regexp.MustCompile(`(?P<Passed>\d+) Passed.*(?P<Failed>\d+) Failed.*Pending`)

	for _, e2eLogPath := range e2eLogFilePaths {
		file, err := os.Open(e2eLogPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			if regexpRanSpecs.MatchString(scanner.Text()) {
				groupToValue, _ := util.GetGroupMapOfRegexMatches(regexpRanSpecs, scanner.Text())
				summary.ExecutedTestcases += util.SilentStrToInt(groupToValue["TestcasesRan"])
				summary.TestsuiteDuration += util.SilentStrToInt(groupToValue["TestSuiteDuration"])
			}
			if regexpPassedFailed.MatchString(scanner.Text()) {
				groupToValue, _ := util.GetGroupMapOfRegexMatches(regexpPassedFailed, scanner.Text())
				summary.SuccessfulTestcases += util.SilentStrToInt(groupToValue["Passed"])
				summary.FailedTestcases += util.SilentStrToInt(groupToValue["Failed"])
				summary.TestsuiteSuccessful = summary.FailedTestcases == 0
			}
		}

		summary.Flaked = summary.FlakedTestcases != 0
	}
	summary.FinishedTime = time.Now()
	summary.StartTime = summary.FinishedTime.Add(time.Second * time.Duration(-summary.TestsuiteDuration))
	file, err := json.MarshalIndent(summary, "", " ")
	if err != nil {
		log.Fatal(errors.Wrapf(err, "Couldn't marshal testsuite summary %s", summary))
	}
	log.Infof("test suite summary: %+v\n", summary)

	summaryFilePath := path.Join(config.ExportPath, TestSummaryFileName)
	if err := ioutil.WriteFile(summaryFilePath, file, 0644); err != nil {
		log.Fatal(errors.Wrapf(err, "Couldn't write %s to file", summaryFilePath))
	}
	return summary
}

type Summary struct {
	ExecutedTestcases   int       `json:"executed_testcases"`
	SuccessfulTestcases int       `json:"successful_testcases"`
	FailedTestcases     int       `json:"failed_testcases"`
	FlakedTestcases     int       `json:"flaked_testcases"`
	Flaked              bool      `json:"individual_testcases_flaked"`
	TestsuiteDuration   int       `json:"duration"`
	TestsuiteSuccessful bool      `json:"successful"`
	DescriptionFile     string    `json:"test_desc_file"`
	StartTime           time.Time `json:"-"`
	FinishedTime        time.Time `json:"-"`
}
