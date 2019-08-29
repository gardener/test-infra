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
	"github.com/gardener/test-infra/pkg/util"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	corescheme "k8s.io/client-go/kubernetes/scheme"
)

type Phase string

const (
	// PhaseRunning is the name of the running phase.
	PhaseRunning Phase = "Running"

	// PHASE_EXIT is the name of the running phase.
	PhaseExit Phase = "Exit"
)

const (
	// TM_KUBECONFIG_PATH is the name of the environment variable that holds the kubeconfigs folder path
	TM_KUBECONFIG_PATH_NAME = "TM_KUBECONFIG_PATH"

	// TM_KUBECONFIG_PATH is the path where kubeconfigs are mounted to tests.
	TM_KUBECONFIG_PATH = "/tmp/tm/kubeconfig"

	// TM_SHARED_PATH_NAME is the name of the environment variable that holds the shared folder path
	TM_SHARED_PATH_NAME = "TM_SHARED_PATH"

	// TM_SHARED_PATH is the path to a shared folder, where content is shared among the workflow steps
	TM_SHARED_PATH = "/tmp/tm/shared"

	// TM_REPO_PATH is the name of the environment variable that holds the repo path
	TM_REPO_PATH_NAME = "TM_REPO_PATH"

	// TM_REPO_PATH is the path where the repo/location is mounted to the tests.
	TM_REPO_PATH = "/src"

	// TM_PHASE_NAME is the name of the environment variable that holds the Test Machinery phase
	TM_PHASE_NAME = "TM_PHASE"

	// TM_EXPORT_PATH is the name of the environment variable that holds the path to the export folder
	TM_EXPORT_PATH_NAME = "TM_EXPORT_PATH"

	// TM_EXPORT_PATH is the path where test results json's are placed to be persisted.
	TM_EXPORT_PATH = "/tmp/tm/export"

	// ExportArtifact is the name of the output artifact where results are stored.
	ExportArtifact = "ExportArtifact"

	// TM_TESTRUN_ID_NAME is the name of the environment variable that holds the current testrun id
	TM_TESTRUN_ID_NAME = "TM_TESTRUN_ID"

	// ConfigMapName is the name of the testmachinery configmap in the cluster
	ConfigMapName = "tm-config"

	// the label and taint of the nodes in the worker pool which are preferably used for workflow pods
	WorkerPoolTaintLabelName = "testload"
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
	Local             bool
	Insecure          bool
	CleanWorkflowPods bool
	GitSecrets        []GitConfig
	S3                *S3Config
}

// S3Config is an object containing the S3 specific configuration
type S3Config struct {
	Endpoint   string
	SSL        bool
	AccessKey  string
	SecretKey  string
	BucketName string
}

// GitSecrets holds all git secrets as defined in the environment variable.
type GitSecrets struct {
	Secrets []GitConfig `yaml:"secrets"`
}

// GitConfig is an object containing config and credentials for a specific github instance.
// It is defined as in cc-config.
type GitConfig struct {
	HttpUrl       string        `yaml:"httpUrl"`
	ApiUrl        string        `yaml:"apiUrl"`
	SkipTls       bool          `yaml:"disable_tls_validation"`
	TechnicalUser TechnicalUser `yaml:"technicalUser"`
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

// String returns the sanitized TestMachinery configuration as formatted string
func (c *TmConfiguration) String() string {
	if c == nil {
		return "<nil>"
	}

	cc := c.Copy()

	if len(cc.GitSecrets) != 0 {
		for i := range cc.GitSecrets {
			if len(cc.GitSecrets[i].TechnicalUser.AuthToken) != 0 {
				cc.GitSecrets[i].TechnicalUser.AuthToken = "--- REDACTED ---"
			}
			if len(cc.GitSecrets[i].TechnicalUser.Password) != 0 {
				cc.GitSecrets[i].TechnicalUser.Password = "--- REDACTED ---"
			}
		}
	}

	if cc.S3 != nil {
		if len(cc.S3.SecretKey) != 0 {
			cc.S3.SecretKey = "--- REDACTED ---"
		}
		if len(cc.S3.AccessKey) != 0 {
			cc.S3.AccessKey = "--- REDACTED ---"
		}
	}

	return util.PrettyPrintStruct(cc)
}

// Copy creates a deep copy of the configuration
func (c *TmConfiguration) Copy() *TmConfiguration {
	if c == nil {
		return nil
	}
	return &TmConfiguration{
		Namespace:         c.Namespace,
		Local:             c.Local,
		Insecure:          c.Insecure,
		CleanWorkflowPods: c.CleanWorkflowPods,
		GitSecrets:        append(make([]GitConfig, 0, len(c.GitSecrets)), c.GitSecrets...),
		S3:                c.S3.Copy(),
	}
}

// Copy creates a deep copy of the s3 config.
func (c *S3Config) Copy() *S3Config {
	if c == nil {
		return nil
	}
	return &S3Config{
		Endpoint:   c.Endpoint,
		SSL:        c.SSL,
		AccessKey:  c.AccessKey,
		SecretKey:  c.SecretKey,
		BucketName: c.BucketName,
	}
}
