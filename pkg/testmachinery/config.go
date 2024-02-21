// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testmachinery

import (
	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	corescheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/gardener/test-infra/pkg/apis/config"
	configinstall "github.com/gardener/test-infra/pkg/apis/config/install"
	tminstall "github.com/gardener/test-infra/pkg/apis/testmachinery/install"
	"github.com/gardener/test-infra/pkg/util"
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

	// TM_GIT_SHA_NAME is the name of the environment variable that holds the current git commit sha of the specified location.
	TM_GIT_SHA_NAME = "TM_GIT_SHA"

	// TM_GIT_REF_NAME is the name of the environment variable that holds the current git ref of the specified location.
	TM_GIT_REF_NAME = "TM_GIT_REF"

	// ConfigMapName is the name of the testmachinery configmap in the cluster
	ConfigMapName = "tm-config"

	// the label and taint of the nodes in the worker pool which are preferably used for workflow pods
	WorkerPoolTaintLabelName = "testload"

	// Name of the argo suspend template name
	PauseTemplateName = "suspend"

	// ArtifactKubeconfigs is the name of the kubeconfigs artifact
	ArtifactKubeconfigs = "kubeconfigs"

	// ArtifactUntrustedKubeconfigs is the name of the kubeconfigs artifacts for untrusted steps
	ArtifactUntrustedKubeconfigs = "untrustedKubeconfigs"

	// ArtifactSharedFolder is the name of the shared folder artifact
	ArtifactSharedFolder = "sharedFolder"
)

const redactedString = "--- REDACTED ---"

// TmConfiguration is an object containing the actual configuration of the Testmachinery
type TmConfiguration struct {
	*config.Configuration
	GitHubSecrets []GitHubInstanceConfig
}

// GitHub represents the github configuration for the testmachinery
type GitHubConfig struct {
	Cache   *config.GitHubCache
	Secrets []GitHubInstanceConfig
}

// GitHub holds all git secrets as defined in the environment variable.
type GitHubSecrets struct {
	Secrets []GitHubInstanceConfig `yaml:"secrets"`
}

// GitHubInstanceConfig is an object containing config and credentials for a specific github instance.
// It is defined as in cc-config.
type GitHubInstanceConfig struct {
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

// ConfigScheme is the core testmachinery scheme
var ConfigScheme = runtime.NewScheme()

func init() {
	testmachinerySchemeBuilder := runtime.NewSchemeBuilder(
		corescheme.AddToScheme,
		tminstall.AddToScheme,
		argov1alpha1.AddToScheme,
	)

	utilruntime.Must(testmachinerySchemeBuilder.AddToScheme(TestMachineryScheme))
	configinstall.Install(ConfigScheme)

	decoder = serializer.NewCodecFactory(TestMachineryScheme).UniversalDecoder()
}

// String returns the sanitized TestMachinery configuration as formatted string
func (c *TmConfiguration) String() string {
	if c == nil {
		return "<nil>"
	}

	cc := c.Copy()

	if len(cc.GitHubSecrets) != 0 {
		for i := range cc.GitHubSecrets {
			if len(cc.GitHubSecrets[i].TechnicalUser.AuthToken) != 0 {
				cc.GitHubSecrets[i].TechnicalUser.AuthToken = redactedString
			}
			if len(cc.GitHubSecrets[i].TechnicalUser.Password) != 0 {
				cc.GitHubSecrets[i].TechnicalUser.Password = redactedString
			}
		}
	}

	if cc.S3 != nil {
		if len(cc.S3.SecretKey) != 0 {
			cc.S3.SecretKey = redactedString
		}
		if len(cc.S3.AccessKey) != 0 {
			cc.S3.AccessKey = redactedString
		}
	}

	if cc.ElasticSearch != nil {
		if len(cc.ElasticSearch.Password) != 0 {
			cc.ElasticSearch.Password = redactedString
		}
	}

	return util.PrettyPrintStruct(cc)
}

// New creates a deep copy of the configuration
func (c *TmConfiguration) Copy() *TmConfiguration {
	if c == nil {
		return nil
	}
	return &TmConfiguration{
		Configuration: c.Configuration.DeepCopy(),
		GitHubSecrets: append(make([]GitHubInstanceConfig, 0, len(c.GitHubSecrets)), c.GitHubSecrets...),
	}
}
