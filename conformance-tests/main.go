package main

import (
	"fmt"
	"os"

	"github.com/gardener/test-infra/conformance-tests/config"
	"github.com/gardener/test-infra/conformance-tests/hydrophone"
	"github.com/gardener/test-infra/pkg/logger"
)

func main() {
	log, err := logger.NewCliLogger()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	config.LogConfig(log.WithName("Config"))

	err = hydrophone.Setup(log.WithName("Setup"))
	if err != nil {
		log.WithName("Setup").Error(err, "Failed to setup Hydrophone environment")
		os.Exit(1)
	}

	err = hydrophone.Run(log.WithName("RunHydrophone"))
	if err != nil {
		log.WithName("RunHydrophone").Error(err, "Failure running conformance tests with Hydrophone")
		//TODO dump shoot logs?
		os.Exit(1)
	}

	// analyze logs

	// publish if necessary

	// fail step if necessary

}
