package main

import (
	"github.com/gardener/test-infra/test/e2etest/config"
	"github.com/gardener/test-infra/test/e2etest/kubetest"
	"github.com/gardener/test-infra/test/e2etest/kubetest/setup"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := setup.Setup(); err != nil {
		log.Fatal(errors.Wrapf(err, "Initial setup invocation failed"))
	}
	desc := kubetest.Generate()
	kubetestResultsPath := kubetest.Run(desc)
	resultSummary := kubetest.Analyze(kubetestResultsPath)
	if config.PublishResultsToTestgrid == true && resultSummary.TestsuiteSuccessful == true {
		kubetest.Publish(config.ExportPath, resultSummary)
	}
}
