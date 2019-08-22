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
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"os"

	"github.com/gardener/test-infra/pkg/util"

	"gopkg.in/yaml.v2"
)

var tmConfig *TmConfiguration

// Setup fetches all configuration values and creates the TmConfiguration.
func Setup() error {
	TESTDEF_PATH = util.Getenv("TESTDEF_PATH", ".test-defs")
	PREPARE_IMAGE = util.Getenv("PREPARE_IMAGE", "eu.gcr.io/gardener-project/gardener/testmachinery/prepare-step:latest")
	BASE_IMAGE = util.Getenv("BASE_IMAGE", "eu.gcr.io/gardener-project/gardener/testmachinery/base-step:0.28.0")

	var gitSecrets GitSecrets
	err := yaml.Unmarshal([]byte(os.Getenv("GIT_SECRETS")), &gitSecrets)
	if err != nil {
		return fmt.Errorf("unable to read git secrets: %s", err.Error())
	}

	tmConfig = &TmConfiguration{
		Namespace:         util.Getenv("TM_NAMESPACE", "default"),
		Insecure:          false,
		CleanWorkflowPods: util.GetenvBool("CLEAN_WORKFLOW_PODS", false),
		GitSecrets:        gitSecrets.Secrets,
		ObjectStore: &ObjectStoreConfig{
			Endpoint:   os.Getenv("S3_ENDPOINT"),
			AccessKey:  os.Getenv("S3_ACCESS_KEY"),
			SecretKey:  os.Getenv("S3_SECRET_KEY"),
			BucketName: os.Getenv("S3_BUCKET_NAME"),
			SSL:        util.GetenvBool("S3_SSL", false),
		},
	}

	return nil
}

// GetConfig returns the current testmachinery configuration
func GetConfig() *TmConfiguration {
	return tmConfig
}

// IsRunInsecure returns if the testmachinery is run locally
func IsRunInsecure() bool {
	return tmConfig.Insecure
}

// GetWorkflowName returns the workflow name of a testruns
func GetWorkflowName(tr *v1beta1.Testrun) string {
	return fmt.Sprintf("%s-wf", tr.Name)
}
