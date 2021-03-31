package kubetest

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	log "github.com/sirupsen/logrus"

	"github.com/gardener/test-infra/integration-tests/e2e/config"
)

const (
	gcsBucket    = "k8s-conformance-gardener"
	gcsProjectID = "gardener"
)

// Publish creates meta files finished.json, started.json in kubetestResultsPath path and uploads them
// and additionally e2e.log and junit_01.xml files to the google cloud storage
func Publish(kubetestResultsPath string, resultSummary Summary) {
	files := make([]string, 0)
	finishedJsonPath := filepath.Join(kubetestResultsPath, "finished.json")
	startedJsonPath := filepath.Join(kubetestResultsPath, "started.json")
	files = append(files,
		filepath.Join(kubetestResultsPath, "junit_01.xml"),
		filepath.Join(kubetestResultsPath, "build-log.txt"),
		finishedJsonPath,
		startedJsonPath,
	)
	createMetadataFiles(startedJsonPath, finishedJsonPath, resultSummary)
	log.Infof("publish to google cloud storage: %v", files)
	uploadTestResultFiles(files)
}

func createMetadataFiles(startedJsonPath, finishedJsonPath string, testSummary Summary) {
	startedJsonContent := []byte(fmt.Sprintf("{\"timestamp\": %d}", testSummary.StartTime.Unix()))
	if err := ioutil.WriteFile(startedJsonPath, startedJsonContent, 06444); err != nil {
		log.Fatal(err)
	}

	testStatus := "FAILURE"
	if testSummary.TestsuiteSuccessful {
		testStatus = "SUCCESS"
	}
	finishedJsonContent := []byte(fmt.Sprintf("{\"timestamp\": %d, \"result\": \"%s\", \"metadata\": {\"shoot-k8s-release\": \"%s\", \"gardener\": \"%s\"}}", testSummary.FinishedTime.Unix(), testStatus, config.K8sRelease, config.GardenerVersion))
	if err := ioutil.WriteFile(finishedJsonPath, finishedJsonContent, 06444); err != nil {
		log.Fatal(err)
	}

}

func uploadTestResultFiles(files []string) {
	_ = os.Setenv("GOOGLE_CLOUD_PROJECT", gcsProjectID)
	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		log.Fatal("environment variable GOOGLE_APPLICATION_CREDENTIALS is not set. Hence no upload to google cloud storage possible.")
	}
	ctx := context.Background()
	// Creates a client.
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	provider := config.CloudProvider
	if config.CloudProvider == "gcp" {
		provider = "gce"
	}
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	for _, fileSourcePath := range files {
		filename := filepath.Base(fileSourcePath)
		if filepath.Base(fileSourcePath) == "junit_01.xml" {
			filename = filepath.Join("artifacts", filename)
		}
		bucketSuffix := ""
		if len(config.TestcaseGroup) == 1 && config.TestcaseGroup[0] == "conformance" {
			bucketSuffix = "-conformance"
		}
		fileTargetPath := fmt.Sprintf("ci-gardener-e2e%s-%s-v%s/%s/%s", bucketSuffix, provider, config.K8sReleaseMajorMinor, timestamp, filename)

		if err := upload(client, gcsBucket, fileSourcePath, fileTargetPath); err != nil {
			switch err {
			case storage.ErrBucketNotExist:
				log.Fatal("Please create the gcsBucket first e.g. with `gsutil mb`")
			default:
				log.Fatal(err)
			}
		}
		log.Infof("uploaded %s, check %s", filename, fmt.Sprintf("https://console.cloud.google.com/storage/browser/%s/%s?project=%s", gcsBucket, filepath.Dir(fileTargetPath), gcsProjectID))
	}
}

func upload(client *storage.Client, bucket, sourcePath, targetPath string) error {
	ctx := context.Background()
	f, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer f.Close()

	wc := client.Bucket(bucket).Object(targetPath).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return nil
}
