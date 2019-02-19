// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
	"flag"
	"os"

	"github.com/gardener/test-infra/pkg/testmachinery/testrun"
	"github.com/gardener/test-infra/pkg/util"
	log "github.com/sirupsen/logrus"
)

var (
	trFilePath string
)

// Connection to remote is needed to validate remote testdefinitions
func main() {
	log.Info("Start Validator")
	flag.Parse()

	tr, err := util.ParseTestrunFromFile(trFilePath)
	if err != nil {
		log.Fatalf("Error parsing testrun: %s", err.Error())
	}

	if err := testrun.Validate(&tr); err != nil {
		log.Fatalf("Invalid Testrun %s: %s", tr.Name, err.Error())
	}

	log.Infof("Testrun %s is valid", tr.Name)
}

func init() {
	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	log.SetOutput(os.Stderr)
	log.SetLevel(log.DebugLevel)

	flag.StringVar(&trFilePath, "testrun", "examples/int-testrun.yaml", "Filepath to the testrun")
}
