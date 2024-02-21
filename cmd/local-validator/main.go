// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	intconfig "github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/testrun"
)

// Connection to remote is needed to validate remote testdefinitions
func main() {
	logger.InitFlags(nil)
	configPath := flag.String("config", "", "Filepath to configuration")
	trFilePath := flag.String("testrun", "examples/int-testrun.yaml", "Filepath to the testrun")
	flag.Parse()

	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}
	decoder := serializer.NewCodecFactory(testmachinery.ConfigScheme).UniversalDecoder()
	config := &intconfig.Configuration{}
	if _, _, err := decoder.Decode(data, nil, config); err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	if err := testmachinery.Setup(config); err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	log, err := logger.New(nil)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
	}

	log.Info("Start Validator")
	log.V(3).Info("test 3")
	log.V(4).Info("test 4")
	log.V(5).Info("test 5")

	tr, err := testmachinery.ParseTestrunFromFile(*trFilePath)
	if err != nil {
		log.Error(err, "unable to parse", "path", *trFilePath)
		os.Exit(1)
	}

	if err, _ := testrun.Validate(log.WithValues("testrun", internalName(tr)), tr); err != nil {
		log.Error(err, "invalid Testrun", "testrun", internalName(tr))
		os.Exit(1)
	}

	log.Info("successfully validated", "testrun", internalName(tr))
}

// internalName determines an internal name that can be the testruns name, generated name or
// if none is defined it returns noName
func internalName(tr *v1beta1.Testrun) string {
	if tr.Name != "" {
		return tr.Name
	}
	if tr.GenerateName != "" {
		return tr.GenerateName
	}
	return "noName"
}
