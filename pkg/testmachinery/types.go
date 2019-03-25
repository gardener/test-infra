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
	argoscheme "github.com/argoproj/argo/pkg/client/clientset/versioned/scheme"
	tmscheme "github.com/gardener/test-infra/pkg/client/testmachinery/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	corescheme "k8s.io/client-go/kubernetes/scheme"
)

const (
	// TM_KUBECONFIG_PATH is the path where kubeconfigs are mounted to tests.
	TM_KUBECONFIG_PATH = "/tmp/tm/kubeconfig"

	// TM_SHARED_PATH is the path to a shared folder, where content is shared among the workflow steps
	TM_SHARED_PATH = "/tmp/tm/shared"

	// TM_REPO_PATH is the path where the repo/location is mounted to the tests.
	TM_REPO_PATH = "/src"

	// PHASE_RUNNING is the name of the running phase.
	PHASE_RUNNING = "Running"

	// TM_EXPORT_PATH is the path where test results json's are placed to be persisted.
	TM_EXPORT_PATH = "/tmp/tm/export"

	// ExportArtifact is the name of the output artifact where results are stored.
	ExportArtifact = "ExportArtifact"

	ConfigMapName = "tm-config"
)

var (
	// TESTDEF_PATH is the path to TestDefinition inside repositories (scripts/integration-tests/argo/tm)
	TESTDEF_PATH string

	// PREPARE_IMAGE is image of the prepare step.
	PREPARE_IMAGE string

	// BASE_IMAGE is used default image if non is specified by TestDefinition.
	BASE_IMAGE string
)

// TmConfiguration is an object containing the actual configuration of the Testmachinery
type TmConfiguration struct {
	Namespace         string
	Insecure          bool
	CleanWorkflowPods bool
	GitSecrets        []*GitConfig
	ObjectStore       *ObjectStoreConfig
}

// ObjectStoreConfig is an object containing the ObjectStore specific configuration
type ObjectStoreConfig struct {
	Endpoint   string
	SSL        bool
	AccessKey  string
	SecretKey  string
	BucketName string
}

// GitSecrets holds all git secrets as defined in the environment variable.
type GitSecrets struct {
	Secrets []*GitConfig `yaml:"secrets"`
}

// GitConfig is an object containing config and credentials for a specific github instance.
// It is defined as in cc-config.
type GitConfig struct {
	HttpUrl       string         `yaml:"httpUrl"`
	ApiUrl        string         `yaml:"apiUrl"`
	SkipTls       bool           `yaml:"disable_tls_validation"`
	TechnicalUser *TechnicalUser `yaml:"technicalUser"`
}

// TechnicalUser holds the actual git credentials.
type TechnicalUser struct {
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	AuthToken string `yaml:"authToken"`
}

// TestMachineryScheme is the scheme used in the testmachinery and testrunner.
var TestMachineryScheme = runtime.NewScheme()

func init() {
	testmachinerySchemeBuilder := runtime.NewSchemeBuilder(
		corescheme.AddToScheme,
		tmscheme.AddToScheme,
		argoscheme.AddToScheme,
	)

	utilruntime.Must(testmachinerySchemeBuilder.AddToScheme(TestMachineryScheme))
}
