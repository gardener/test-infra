// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

func init() {
	logger.InitFlags(nil)
}

func main() {
	flag.Parse()
	log, err := logger.NewCliLogger()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if len(os.Args) == 1 {
		log.Error(nil, "no arguments specified. A path to the config file has to be defined")
		os.Exit(1)
	}

	configFilePath := os.Args[1]

	cfg, err := readConfigFile(configFilePath)
	if err != nil {
		log.Error(err, "unable to read config", "path", configFilePath)
		os.Exit(1)
	}

	repoBasePath := os.Getenv(testmachinery.TM_REPO_PATH_NAME)
	if err := runPrepare(log, cfg, repoBasePath); err != nil {
		log.Error(err, "error running prepare")
		os.Exit(1)
	}

	log.Info("Prepare finished")
}
