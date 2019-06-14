package kubetest

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/gardener/test-infra/test/e2etest/config"
	"github.com/gardener/test-infra/test/e2etest/util"
	"github.com/gardener/test-infra/test/e2etest/util/sets"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	FALSE_POSITIVES_DESC_FILE = "false_positives.json"
	SKIP_DESC_FILE            = "skip.json"
	GENERATED_RUN_DESC_FILE   = "generated_tests_to_run.txt"
	SUCCESS                   = "success"
	FAILURE                   = "failure"
)

var falsePositiveDescPath = filepath.Join(config.DescriptionsPath, FALSE_POSITIVES_DESC_FILE)
var GeneratedRunDescPath = filepath.Join(config.TmpDir, GENERATED_RUN_DESC_FILE)
var skipDescPath = filepath.Join(config.DescriptionsPath, SKIP_DESC_FILE)

func Generate() (desc string) {
	testcasesToRun := sets.NewStringSet()
	allE2eTestcases := getAllE2eTestCases()

	if config.DescriptionFilePath != "" {
		testcasesFromDescriptionFile := getTestcaseNamesFromDesc(config.DescriptionFilePath)
		testcasesToRun = allE2eTestcases.GetSetOfMatching(testcasesFromDescriptionFile)
	}

	if config.RetestFlaggedOnly {
		var testcasesFromDescriptionFile = sets.NewStringSet()
		testcasesFromDescriptionFile.Insert("[Conformance]")
		testcasesToRun = allE2eTestcases.GetSetOfMatching(testcasesFromDescriptionFile)
	}

	if !config.IgnoreFalsePositiveList {
		falsePositiveTestcases := getTestcaseNamesFromDesc(falsePositiveDescPath)
		for falsePositiveTestcase := range falsePositiveTestcases {
			testcasesToRun.Delete(falsePositiveTestcase)
		}
	}

	if !config.IgnoreSkipList {
		skipTestcases := getTestcaseNamesFromDesc(skipDescPath)
		for skipTestcase := range skipTestcases {
			testcasesToRun.DeleteMatching(skipTestcase)
		}
	}

	if config.IncludeUntrackedTests {
		untrackedTestcases := allE2eTestcases
		trackedTestcases := sets.NewStringSet()
		descFiles := util.GetFilesByPattern(config.DescriptionsPath, `\.json`)
		for _, descFile := range descFiles {
			trackedTestcases.Union(getTestcaseNamesFromDesc(descFile))
		}
		for trackedTestcase := range trackedTestcases {
			untrackedTestcases.DeleteMatching(trackedTestcase)
		}
		testcasesToRun.Union(untrackedTestcases)
	}

	if len(testcasesToRun) == 0 {
		log.Fatal("No testcases found to run.")
	}

	// write testcases to run to file
	if err := writeLinesToFile(testcasesToRun, GeneratedRunDescPath); err != nil {
		log.Fatal(errors.Wrapf(err, "Couldn't save testcasesToRun as file in %s", GeneratedRunDescPath))
	}
	return GeneratedRunDescPath
}

func getTestcaseNamesFromDesc(descPath string) sets.StringSet {
	testcasesOfCurrentProvider := sets.NewStringSet()
	testcases := UnmarshalDescription(descPath)
	for _, testcase := range testcases {
		if len(testcase.ExcludedProviders) != 0 && len(testcase.OnlyProviders) != 0 {
			log.Warn("fields excluded and only of description file testcase, are not allowed to be defined both at the same time. Skipping testcase: %s", testcase.Name)
			continue
		}
		// check
		excludedExplicitly := util.Contains(testcase.ExcludedProviders, config.CloudProvider)
		excludedImplicitly := len(testcase.OnlyProviders) != 0 && !util.Contains(testcase.OnlyProviders, config.CloudProvider)
		retestActiveForThisProviderAndTest := config.RetestFlaggedOnly && util.Contains(testcase.Retest, config.CloudProvider)
		if !excludedExplicitly && !excludedImplicitly && !config.RetestFlaggedOnly || retestActiveForThisProviderAndTest {
			testcasesOfCurrentProvider.Insert(testcase.Name)
		}
	}
	return testcasesOfCurrentProvider
}

func getAllE2eTestCases() sets.StringSet {
	allTestcases := sets.NewStringSet()
	resultsPath := DryRun()
	defer os.RemoveAll(resultsPath)
	junitPaths := util.GetFilesByPattern(resultsPath, JunitXmlFileNamePattern)
	if len(junitPaths) > 1 {
		log.Fatalf("Found multiple junit.xml files after dry run of kubetest in %s. Expected only one.", resultsPath)
	}
	if len(junitPaths) == 0 {
		log.Fatalf("No junit file has been created during kubetest dry run.", resultsPath)
	}
	junitXml, err := UnmarshalJunitXMLResult(junitPaths[0])
	if err != nil {
		log.Fatal(errors.Wrapf(err, "Couldn't unmarshal junit.xml %s", junitPaths[0]))
	}

	// get testcase names of all not skipped testcases
	for _, testcase := range junitXml.Testcases {
		if !testcase.Skipped {
			allTestcases.Insert(testcase.Name)
		}
	}
	return allTestcases
}

func UnmarshalDescription(descPath string) []TestcaseDesc {
	var testcases []TestcaseDesc
	descFile, err := ioutil.ReadFile(descPath)
	if err != nil {
		log.Fatal(errors.Wrapf(err, "Couldn't read file %s: %s", descPath, descFile))
	}
	if err = json.Unmarshal(descFile, &testcases); err != nil {
		log.Fatal(errors.Wrapf(err, "Couldn't unmarshal %s: %s", descPath, descFile))
	}
	return testcases
}

func UnmarshalJunitXMLResult(junitXmlPath string) (junitXml JunitXMLResult, err error) {
	var xmlResult JunitXMLResult
	junitXML, err := ioutil.ReadFile(junitXmlPath)
	if err != nil {
		return xmlResult, err
	}
	if err = xml.Unmarshal(junitXML, &xmlResult); err != nil {
		return xmlResult, err
	}

	xmlResult.CalculateAdditionalFields()
	return xmlResult, nil
}

// writeLinesToFile writes the lines to the given file.
func writeLinesToFile(lines sets.StringSet, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			log.Warn(errors.Wrapf(err, "Couldn't write text '%s' to file %s", line, file))
		}
	}
	return w.Flush()
}

type TestcaseDesc struct {
	Name              string   `json:"testcase"`
	ExcludedProviders []string `json:"exclude,omitempty"`
	OnlyProviders     []string `json:"only,omitempty"`
	Retest            []string `json:"retest,omitempty"`
}
