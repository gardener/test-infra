package kubetest

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gardener/test-infra/integration-tests/e2e/config"
	"github.com/gardener/test-infra/integration-tests/e2e/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	E2eLogFileNamePattern   = "e2e.log$"
	JunitXmlFileNamePattern = `junit_\d+.xml$`
	TestSummaryFileName     = "test_summary.json"
	MergedJunitXmlFile      = "junit_01.xml"
	MergedE2eLogFile        = "build-log.txt"
)

var mergedJunitXmlFilePath = filepath.Join(config.ExportPath, MergedJunitXmlFile)
var MergedE2eLogFilePath = filepath.Join(config.ExportPath, MergedE2eLogFile)

// Analyze analyzes junit.xml files and e2e.log files, which are dumped by kubetest and provides a resulting
// test suite summary and results for each testcase individually. These results are then written
// to the export dir as files.
func Analyze(kubetestResultsPath string) Summary {
	log.Infof("Analyze e2e.log and junit.xml files in %s", kubetestResultsPath)

	e2eLogFilePaths := util.GetFilesByPattern(kubetestResultsPath, E2eLogFileNamePattern)
	summary, err := analyzeE2eLogs(e2eLogFilePaths)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "results analysis failed at e2e.log analysis"))
	}

	junitXMLFilePaths := util.GetFilesByPattern(kubetestResultsPath, JunitXmlFileNamePattern)
	if err := analyzeJunitXMLsEnrichSummary(junitXMLFilePaths, &summary); err != nil {
		log.Fatal(errors.Wrapf(err, "results analysis failed at junit.xml analysis"))
	}

	log.Infof("test suite summary: %+v\n", summary)
	writeSummaryToFile(summary)
	log.Infof("Check out result files in %s", config.ExportPath)

	return summary
}

func writeSummaryToFile(summary Summary) {
	file, err := json.Marshal(summary)
	file = append([]byte("{\"index\": {\"_index\": \"e2e_testsuite\", \"_type\": \"_doc\"}}\n"), file...)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "couldn't marshal testsuite summary %s", summary))
	}
	summaryFilePath := path.Join(config.ExportPath, TestSummaryFileName)
	if err := ioutil.WriteFile(summaryFilePath, file, 0644); err != nil {
		log.Fatal(errors.Wrapf(err, "Couldn't write %s to file", summaryFilePath))
	}
}

func analyzeJunitXMLsEnrichSummary(junitXMLFilePaths []string, summary *Summary) error {
	mergedJunitXmlResult := &JunitXMLResult{DurationInt: summary.TestsuiteDuration}
	var testcases []TestcaseResult
	failureOccurrences := make(map[string]int)  // map of testcases that failed at least once
	succeededTestcases := make(map[string]bool) // map of testcases that succeeded at east once
	skippedTestcases := make(map[string]TestcaseResult)
	uniqueJunitTestcases := make(map[string]TestcaseResult)

	for _, junitXMLPath := range junitXMLFilePaths {
		junitXml, err := unmarshalJUnitFromFile(junitXMLPath)
		junitXml.CalculateAdditionalFields()
		if err != nil {
			return err
		}
		mergedJunitXmlResult.Errors += junitXml.Errors
		// scatter testcases in groups succeeded, failed, skipped
		for _, newTestcase := range junitXml.Testcases {
			_, existsInFailureOccurences := failureOccurrences[newTestcase.Name]
			_, existsInSucceededTestcases := succeededTestcases[newTestcase.Name]
			if newTestcase.Skipped {
				if !existsInFailureOccurences && !existsInSucceededTestcases {
					skippedTestcases[newTestcase.Name] = newTestcase
				}
				// skipped testcases are appended later, to avoid duplicates. Therefore continue with next testcase
				continue
			}
			testcases = append(testcases, newTestcase) // collect testcases that are either failed or succeeded
			delete(skippedTestcases, newTestcase.Name) // if testcase was indexed as skipped from previous junit xml files, remove element
			if newTestcase.Successful {
				succeededTestcases[newTestcase.Name] = true
				uniqueJunitTestcases[newTestcase.Name] = newTestcase
			} else {
				if existsInFailureOccurences {
					failureOccurrences[newTestcase.Name]++
				} else {
					uniqueJunitTestcases[newTestcase.Name] = newTestcase
					failureOccurrences[newTestcase.Name] = 1
				}
			}
		}
	}

	if err := junitXMLTestcasesToJSON(&testcases, &failureOccurrences, &succeededTestcases); err != nil {
		return err
	}

	for _, testcase := range uniqueJunitTestcases {
		mergedJunitXmlResult.Testcases = append(mergedJunitXmlResult.Testcases, testcase)
	}
	for _, testcase := range skippedTestcases {
		mergedJunitXmlResult.Testcases = append(mergedJunitXmlResult.Testcases, testcase)
	}
	addAdditionalInfoToSummary(summary, &failureOccurrences, &succeededTestcases)
	mergedJunitXmlResult.DurationFloat = float32(summary.TestsuiteDuration)
	mergedJunitXmlResult.DurationInt = summary.TestsuiteDuration
	mergedJunitXmlResult.FailedTests = summary.FailedTestcases
	mergedJunitXmlResult.ExecutedTests = summary.ExecutedTestcases
	mergedJunitXmlResult.SuccessfulTests = summary.SuccessfulTestcases
	if err := saveJunitXmlToFile(junitXMLFilePaths, mergedJunitXmlResult); err != nil {
		return err
	}
	return nil
}

func junitXMLTestcasesToJSON(testcases *[]TestcaseResult, failureOccurrences *map[string]int, succeededTestcases *map[string]bool) error {
	uniqueTestcases := make(map[string]TestcaseResult)
	for _, testcase := range *testcases {
		if testcase.Skipped {
			// skipped testcases are not saved as json files
			continue
		}
		if _, ok := uniqueTestcases[testcase.Name]; ok {
			// continue if testcase has been already registered
			continue
		}
		failuresCount, everFailed := (*failureOccurrences)[testcase.Name]
		if everFailed && testcase.Successful {
			// since we need the failure description, we skip the successful testcase execution and only save the failed one later
			continue
		}
		_, everSucceeded := (*succeededTestcases)[testcase.Name]
		if !testcase.Successful && everSucceeded {
			// if testcase failed but was successful at some point in time, then it flaked
			testcase.Flaked = failuresCount
			testcase.Successful = true
			testcase.Status = "success"
			testcase.StatusShort = "S"
			testcase.SuccessRate = 100
			uniqueTestcases[testcase.Name] = testcase
		} else {
			// successful or failed testcase
			uniqueTestcases[testcase.Name] = testcase
		}
	}
	for _, newTestcase := range uniqueTestcases {
		if err := writeTestcaseToJSONFile(newTestcase); err != nil {
			return err
		}
	}
	return nil
}

func unmarshalJUnitFromFile(junitXMLPath string) (JunitXMLResult, error) {
	file, err := os.Open(junitXMLPath)
	if err != nil {
		return JunitXMLResult{}, err
	}
	defer file.Close()
	junitXml, err := UnmarshalJunitXMLResult(file.Name())
	if err != nil {
		return JunitXMLResult{}, errors.Wrapf(err, "Couldn't unmarshal %s", file.Name())
	}
	return junitXml, nil
}

func writeTestcaseToJSONFile(testcase TestcaseResult) error {
	testcaseJSON, err := json.Marshal(testcase)
	if err != nil {
		return errors.Wrapf(err, "Couldn't marshal testsuite summary %s", testcaseJSON)
	}
	testcaseJSON = append([]byte("{\"index\": {\"_index\": \"e2e_testcase\", \"_type\": \"_doc\"}}\n"), testcaseJSON...)

	jsonFileName := fmt.Sprintf("test-%s.json", strconv.FormatInt(time.Now().UnixNano(), 10))
	testcaseJsonFilePath := path.Join(config.ExportPath, jsonFileName)
	if err := ioutil.WriteFile(testcaseJsonFilePath, testcaseJSON, 0644); err != nil {
		return errors.Wrapf(err, "Couldn't write %s to file", testcaseJsonFilePath)
	}
	return nil
}

func addAdditionalInfoToSummary(summary *Summary, failureOccurrences *map[string]int, succeededTestcases *map[string]bool) {
	for testcaseName, failureOccurrence := range *failureOccurrences {
		if (*succeededTestcases)[testcaseName] {
			// testcase succeeded at least once
			summary.FlakedTestcases += failureOccurrence
		} else {
			// testcase has failed in all attempts
			summary.FailedTestcaseNames = append(summary.FailedTestcaseNames, testcaseName)
		}
	}
	summary.ExecutedTestcases = len(summary.FailedTestcaseNames) + len(*succeededTestcases)
	summary.SuccessfulTestcases = len(*succeededTestcases)
	summary.FailedTestcases = len(summary.FailedTestcaseNames)
	summary.TestsuiteSuccessful = summary.FailedTestcases == 0
	if summary.FlakedTestcases != 0 {
		summary.Flaked = true
	}
}

func saveJunitXmlToFile(junitXMLFilePaths []string, mergedJunitXmlResult *JunitXMLResult) error {
	if len(junitXMLFilePaths) == 1 {
		// if there is only one single junit xml file, we can use the original file as output
		// this is especially the case if conformance tests are executed
		if _, err := util.Copy(junitXMLFilePaths[0], mergedJunitXmlFilePath); err != nil {
			return err
		}
		return nil
	}
	output, err := xml.MarshalIndent(mergedJunitXmlResult, "  ", "    ")
	if err != nil {
		return err
	}
	output = append([]byte(xml.Header), output...)

	if err = ioutil.WriteFile(mergedJunitXmlFilePath, output, 0644); err != nil {
		return err
	}
	return nil
}

func analyzeE2eLogs(e2eLogFilePaths []string) (Summary, error) {
	mergeE2eLogFiles(MergedE2eLogFilePath, e2eLogFilePaths)

	emptySummary := Summary{DescriptionFile: config.DescriptionFile}
	summary := emptySummary
	regexpGinkgoRanIn := regexp.MustCompile(`Ginkgo ran \d+ suite in (?P<TestSuiteDuration>.+)`)

	for _, e2eLogPath := range e2eLogFilePaths {
		file, err := os.Open(e2eLogPath)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		for lineByte := range util.ReadLinesFromFile(file) {
			lineString := string(lineByte)
			if regexpGinkgoRanIn.MatchString(lineString) {
				log.Info(lineString)
				groupToValue := util.GetGroupMapOfRegexMatches(regexpGinkgoRanIn, lineString)
				duration, err := time.ParseDuration(groupToValue["TestSuiteDuration"])
				if err != nil {
					return summary, err
				}
				summary.TestsuiteDuration += int(duration.Seconds())
			}
		}
		if summary.TestsuiteDuration == 0 {
			contentBytes, err := ioutil.ReadFile(e2eLogPath)
			if err != nil {
				log.Fatal(err)
			}
			log.Fatalf("Wasn't able to interpret e2e.log. Got only zero values. Check e2e.log output:\n%s", string(contentBytes))
		}
	}
	summary.ExecutionGroup = strings.Join(config.TestcaseGroup, ",")
	summary.FinishedTime = time.Now()
	summary.StartTime = summary.FinishedTime.Add(time.Second * time.Duration(-summary.TestsuiteDuration))

	return summary, nil
}

func convertValuesToInt(m map[string]string) (map[string]int, error) {
	convertedMap := make(map[string]int, len(m))
	for key, value := range m {
		if key == "" {
			continue // ignore fields without a key
		}
		convertedValue, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		convertedMap[key] = convertedValue
	}
	return convertedMap, nil
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
	if len(e2eLogFilePaths) == 1 {
		log.Infof("copied %s file to %s/%s", e2eLogFilePaths[0], dst, MergedE2eLogFile)
	} else {
		log.Infof("merged %o e2e log files to %s%s", len(e2eLogFilePaths), dst, MergedE2eLogFile)
	}
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
	FailedTestcaseNames []string  `json:"failed_testcase_names"`
}
