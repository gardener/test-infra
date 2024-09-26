// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/conformance-tests/config"
)

const (
	startedFileName  = "started.json"
	finishedFileName = "finished.json"
	junitXMLFileName = "junit_01.xml"
	e2eFileName      = "e2e.log"
)

func Publish(log logr.Logger) error {
	err := createMetadataFiles(log.WithName("JunitParser"))
	if err != nil {
		return err
	}

	filesToUpload := []string{
		filepath.Join(config.ExportPath, startedFileName),
		filepath.Join(config.ExportPath, finishedFileName),
		filepath.Join(config.ExportPath, junitXMLFileName),
		filepath.Join(config.ExportPath, e2eFileName),
	}

	err = uploadResultsToBucket(filesToUpload)
	if err != nil {
		//todo log
		return err
	}
	return nil
}

func createMetadataFiles(log logr.Logger) error {
	testsuiteFinishTime, testsuiteStartTime, err := parseJunit(filepath.Join(config.ExportPath, junitXMLFileName))
	if err != nil {
		log.Error(err, "Failure parsing junit_01.xml file to extract start time and duration")
		return err
	}

	startedContent := []byte(fmt.Sprintf("{\"timestamp\": %d}", testsuiteStartTime.Unix()))
	if err := os.WriteFile(filepath.Join(config.ExportPath, startedFileName), startedContent, 06444); err != nil {
		return err
	}

	finishedContent := []byte(fmt.Sprintf("{\"timestamp\": %d, \"result\": \"SUCCESS\", \"metadata\": {\"shoot-k8s-release\": \"%s\", \"gardener\": \"%s\"}}", testsuiteFinishTime.Unix(), config.K8sRelease, config.GardenerVersion))
	if err := os.WriteFile(filepath.Join(config.ExportPath, finishedFileName), finishedContent, 06444); err != nil {
		return err
	}

	return nil
}

func uploadResultsToBucket(files []string) error {
	//todo add upload logic
	return nil
}
