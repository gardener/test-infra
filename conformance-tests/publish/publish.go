// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
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

	k8sReleaseMajorMinor := string(regexp.MustCompile(`^(\d+\.\d+)`).FindSubmatch([]byte(config.K8sRelease))[1])
	return uploadResultsToBucket(log.WithName("UploadToBucket"), filesToUpload, k8sReleaseMajorMinor)
}

func createMetadataFiles(log logr.Logger) error {
	testsuiteFinishTime, testsuiteStartTime, err := parseJunit(filepath.Join(config.ExportPath, junitXMLFileName))
	if err != nil {
		log.Error(err, "Failure parsing junit_01.xml file to extract start time and duration")
		return err
	}

	startedContent := []byte(fmt.Sprintf("{\"timestamp\": %d}", testsuiteStartTime.Unix()))
	err = os.WriteFile(filepath.Join(config.ExportPath, startedFileName), startedContent, 0600)
	if err != nil {
		log.Error(err, "Failed to write started.json file")
		return err
	}

	finishedContent := []byte(fmt.Sprintf("{\"timestamp\": %d, \"result\": \"SUCCESS\", \"metadata\": {\"shoot-k8s-release\": \"%s\", \"gardener\": \"%s\"}}", testsuiteFinishTime.Unix(), config.K8sRelease, config.GardenerVersion))
	err = os.WriteFile(filepath.Join(config.ExportPath, finishedFileName), finishedContent, 0600)
	if err != nil {
		log.Error(err, "Failed to write finished.json file")
		return err
	}

	return nil
}

func uploadResultsToBucket(log logr.Logger, files []string, k8sReleaseMajorMinor string) error {
	err := os.Setenv("GOOGLE_CLOUD_PROJECT", config.GcsProjectID)
	if err != nil {
		log.Error(err, "Cannot set GOOGLE_CLOUD_PROJECT env variable")
		return err
	}

	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Error(err, "Failed to construct a storage client")
		return err
	}
	provider := config.CloudProvider
	if config.CloudProvider == "gcp" {
		provider = "gce"
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	for _, sourceFilePath := range files {
		fileName := filepath.Base(sourceFilePath)
		if filepath.Base(sourceFilePath) == junitXMLFileName {
			fileName = filepath.Join("artifacts", junitXMLFileName)
		}
		targetFilePath := fmt.Sprintf("ci-gardener-e2e-conformance-%s-v%s/%s/%s", provider, k8sReleaseMajorMinor, timestamp, fileName)

		err := upload(gcsClient, config.GcsBucket, sourceFilePath, targetFilePath)
		if err != nil {
			log.Error(err, "Failed to upload file bucket", "sourceFilePath", sourceFilePath, "targetFilePath", targetFilePath)
			return err
		}
		log.Info("File upload successful", fileName, fmt.Sprintf("https://console.cloud.google.com/storage/browser/%s/%s?project=%s", config.GcsBucket, filepath.Dir(targetFilePath), config.GcsProjectID))
	}

	return nil
}

func upload(client *storage.Client, bucket, sourcePath, targetPath string) error {
	ctx := context.Background()
	f, err := os.Open(filepath.Clean(sourcePath))
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Printf("Failed to close file %s: %v\n", f.Name(), err)
		}
	}(f)

	wc := client.Bucket(bucket).Object(targetPath).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return nil
}
