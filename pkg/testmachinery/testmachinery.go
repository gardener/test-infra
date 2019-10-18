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

package testmachinery

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/version"
	"io/ioutil"
	"os"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/util"

	"gopkg.in/yaml.v2"
)

var (
	githubSecretsPath string
	objectStoreConfig S3Config
)

var tmConfig = TmConfiguration{
	Local:             false,
	Insecure:          false,
	Namespace:         "default",
	CleanWorkflowPods: false,
	GitSecrets:        make([]GitConfig, 0),
	S3:                &objectStoreConfig,
}

// Setup fetches all configuration values and creates the TmConfiguration.
func Setup() error {
	var err error
	tmConfig.GitSecrets, err = readSecretsFromFile(githubSecretsPath)
	if err != nil {
		return err
	}

	// if no endpoint is defined we assume that no cleanup should happen
	// this should only happen in local environments
	if tmConfig.Local && len(objectStoreConfig.Endpoint) == 0 {
		tmConfig.S3 = nil
	}
	if err := ValidateS3Config(tmConfig.S3); err != nil {
		return err
	}

	return nil
}

func InitFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	flagset.StringVar(&TESTDEF_PATH, "testdef-path", util.Getenv("TESTDEF_PATH", ".test-defs"),
		"Set repository path where the Test Machinery should search for testdefinition")
	flagset.StringVar(&PREPARE_IMAGE, "prepare-image", util.Getenv("PREPARE_IMAGE", fmt.Sprintf("eu.gcr.io/gardener-project/gardener/testmachinery/prepare-step:%s", version.Get().GitVersion)),
		"Set the prepare image that is used in the prepare and postprepare step")
	flagset.StringVar(&BASE_IMAGE, "base-image", util.Getenv("BASE_IMAGE", fmt.Sprintf("eu.gcr.io/gardener-project/gardener/testmachinery/base-step:%s", version.Get().GitVersion)),
		"Set the base image that is used as the default image if a TestDefinition does not define a image")

	flag.BoolVar(&tmConfig.Local, "local", false, "The controller runs outside of a cluster.")
	flagset.BoolVar(&tmConfig.Insecure, "insecure", tmConfig.Insecure,
		"Enable insecure mode. The test machinery runs in insecure mode which means that local testdefs are allowed and therefore hostPaths are mounted.")
	flagset.StringVar(&tmConfig.Namespace, "namespace", util.Getenv("TM_NAMESPACE", tmConfig.Namespace),
		"Set the namespace of the testmachinery")
	flagset.BoolVar(&tmConfig.CleanWorkflowPods, "enable-pod-gc", util.GetenvBool("CLEAN_WORKFLOW_PODS", tmConfig.CleanWorkflowPods),
		"Enable garbage collection of pods after a testrun has finished")

	flagset.StringVar(&githubSecretsPath, "github-secrets-path", "",
		"Path to the github secrets configuration")
	flagset.StringVar(&objectStoreConfig.Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"),
		"Set the s3 object storage endpoint")
	flagset.StringVar(&objectStoreConfig.AccessKey, "s3-access-key", os.Getenv("S3_ACCESS_KEY"),
		"Set the s3 object storage access key")
	flagset.StringVar(&objectStoreConfig.SecretKey, "s3-secret-key", os.Getenv("S3_SECRET_KEY"),
		"Set the s3 object storage secret key")
	flagset.StringVar(&objectStoreConfig.BucketName, "s3-bucket", os.Getenv("S3_BUCKET_NAME"),
		"Set the s3 bucket")
	flagset.BoolVar(&objectStoreConfig.SSL, "s3-ssl", util.GetenvBool("S3_SSL", objectStoreConfig.SSL),
		"Enable sll communication to s3 storage")
}

// GetConfig returns the current testmachinery configuration
func GetConfig() *TmConfiguration {
	return &tmConfig
}

// IsRunInsecure returns if the testmachinery is run locally
func IsRunInsecure() bool {
	return tmConfig.Insecure
}

// GetWorkflowName returns the workflow name of a testruns
func GetWorkflowName(tr *v1beta1.Testrun) string {
	return fmt.Sprintf("%s-wf", tr.Name)
}

func readSecretsFromFile(path string) ([]GitConfig, error) {
	if len(path) == 0 {
		return make([]GitConfig, 0), nil
	}
	if _, err := os.Stat(githubSecretsPath); err != nil {
		return nil, errors.Wrapf(err, "file %s does not exist", path)
	}
	rawSecrets, err := ioutil.ReadFile(githubSecretsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read file from %s", githubSecretsPath)
	}
	gitSecrets := GitSecrets{}
	err = yaml.Unmarshal(rawSecrets, &gitSecrets)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse git secrets")
	}
	if len(gitSecrets.Secrets) == 0 {
		return nil, errors.New("git secrets are emtpy")
	}
	return gitSecrets.Secrets, nil
}
