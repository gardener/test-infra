// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/gardener/test-infra/conformance-tests/config"
	"github.com/gardener/test-infra/conformance-tests/hydrophone"
	"github.com/gardener/test-infra/conformance-tests/util"
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
		log.WithName("RunHydrophone").Error(err, "Conformance tests with Hydrophone did not finish successful")
		os.Exit(1)
	}

	if config.PublishResultsToTestgrid {
		// analyze logs (write start/end files, gg version, ...)
		err = util.Publish(log.WithName("PublishResults"))
		if err != nil {
			log.WithName("PublishResults").Error(err, "Failed to publish test results")
			os.Exit(1)
		}
	}

	log.Info("Finished conformance test setp successfully")
}
