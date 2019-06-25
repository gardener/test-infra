package kubetest

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/gardener/test-infra/test/e2etest/config"
	"github.com/gardener/test-infra/test/e2etest/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	E2eLogFileNamePattern   = "e2e.log$"
	JunitXmlFileNamePattern = `junit_\d+.xml$`
	TestSummaryFileName     = "test_summary.json"
	MergedJunitXmlFile      = "junit_01.xml"
	MergedE2eLogFile        = "e2e.log"
)

var mergedJunitXmlFilePath = filepath.Join(config.ExportPath, MergedJunitXmlFile)
var MergedE2eLogFilePath = filepath.Join(config.ExportPath, MergedE2eLogFile)

// Analyze analyzes junit.xml files and e2e.log files, which are dumped by kubetest and provides a resulting test suite summary and results for each testcase individually. These results are then written to the export dir as files.
func Analyze(kubetestResultsPath string) Summary {
	log.Info("Analyze e2e.log and junit.xml files")
	e2eLogFilePaths := util.GetFilesByPattern(kubetestResultsPath, E2eLogFileNamePattern)
	summary := analyzeE2eLogs(e2eLogFilePaths)
	junitXMLFilePaths := util.GetFilesByPattern(kubetestResultsPath, JunitXmlFileNamePattern)
	analyzeJunitXMLs(junitXMLFilePaths, summary.TestsuiteDuration)
	log.Infof("Check out result files in %s", kubetestResultsPath)
	return summary
}

func analyzeJunitXMLs(junitXMLFilePaths []string, durationSec int) {
	var mergedJunitXmlResult = &JunitXMLResult{FailedTests: 0, ExecutedTests: 0, DurationFloat: 0, SuccessfulTests: 0, DurationInt: durationSec}
	testcaseNameToTestcase := make(map[string]TestcaseResult)
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
		mergedJunitXmlResult.FailedTests += junitXml.FailedTests
		mergedJunitXmlResult.ExecutedTests += junitXml.ExecutedTests
		mergedJunitXmlResult.SuccessfulTests += junitXml.SuccessfulTests
		for _, testcase := range junitXml.Testcases {
			if testcase.Skipped {
				if _, ok := testcaseNameToTestcase[testcase.Name]; !ok {
					testcaseNameToTestcase[testcase.Name] = testcase
				}
				continue
			}
			testcaseNameToTestcase[testcase.Name] = testcase
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
	for _, testcase := range testcaseNameToTestcase {
		mergedJunitXmlResult.Testcases = append(mergedJunitXmlResult.Testcases, testcase)
	}
	saveJunitXmlToFile(mergedJunitXmlResult)
}

func saveJunitXmlToFile(mergedJunitXmlResult *JunitXMLResult) {
	output, err := xml.MarshalIndent(mergedJunitXmlResult, "  ", "    ")
	if err != nil {
		fmt.Printf("error: %v\n", err)
	}
	output = append([]byte(xml.Header), output...)

	file, _ := os.Create(mergedJunitXmlFilePath)
	defer file.Close()
	if _, err = file.Write(output); err != nil {
		log.Fatal(err)
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
	summary.ExecutionGroup = strings.Join(config.TestcaseGroup, ",")
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

	mergeE2eLogFiles(MergedE2eLogFilePath, e2eLogFilePaths)
	return summary
}

func mergeE2eLogFiles(dst string, e2eLogFilePaths []string) {
	resultFile, _ := os.Create(dst)

	for _, e2eLogFile := range e2eLogFilePaths {
		fileToAppend, err := os.Open(e2eLogFile)
		if err != nil {
			log.Fatalln("failed to open file %s for reading:", e2eLogFile, err)
		}
		defer fileToAppend.Close()

		if _, err := io.Copy(resultFile, fileToAppend); err != nil {
			log.Fatalln("failed to append file %s to file %s:", fileToAppend, resultFile, err)
		}
	}
	log.Infof("Merged %o e2e log files to %s", len(e2eLogFilePaths), dst)
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
	ExecutionGroup      string    `json:"execution_group"`
}
