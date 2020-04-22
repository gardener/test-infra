// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery"
	flag "github.com/spf13/pflag"
	"os"
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
