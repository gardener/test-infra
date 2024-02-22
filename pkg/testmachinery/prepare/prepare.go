// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package prepare

import (
	"encoding/json"
	"fmt"
	"net/url"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/locations/location"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/util"
)

// New creates the TM prepare step
// The step clones all needed github config and outputs these repos as argo artifacts with the name "repoOwner-repoName-revision".
func New(name string, addGlobalInput, addGlobalOutput bool) (*Definition, error) {
	td := testdefinition.NewEmpty()
	td.Info = &tmv1beta1.TestDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	td.Template = &argov1.Template{
		Name: fmt.Sprintf("%s-%s", name, util.RandomString(5)),
		Metadata: argov1.Metadata{
			Annotations: map[string]string{
				common.AnnotationTestDefName: "Prepare",
				common.AnnotationSystemStep:  "true",
			},
		},
		ActiveDeadlineSeconds: &testdefinition.DefaultActiveDeadlineSeconds,
		Container: &corev1.Container{
			Image:   testmachinery.PrepareImage(),
			Command: []string{"/tm/prepare", PrepareConfigPath},
			Env: []corev1.EnvVar{
				{
					Name:  testmachinery.TM_PHASE_NAME,
					Value: "{{inputs.parameters.phase}}",
				},
				{
					Name:  testmachinery.TM_REPO_PATH_NAME,
					Value: testmachinery.TM_REPO_PATH,
				},
				{
					Name:  testmachinery.TM_KUBECONFIG_PATH_NAME,
					Value: testmachinery.TM_KUBECONFIG_PATH,
				},
				{
					Name:  testmachinery.TM_SHARED_PATH_NAME,
					Value: testmachinery.TM_SHARED_PATH,
				},
			},
		},
		Inputs: argov1.Inputs{
			Parameters: []argov1.Parameter{
				{Name: "phase"},
			},
		},
	}
	prepare := &Definition{td, addGlobalInput, Config{
		Directories:  []string{testmachinery.TM_KUBECONFIG_PATH, testmachinery.TM_SHARED_PATH},
		Repositories: make(map[string]*Repository),
	}}

	if err := prepare.addNetrcFile(); err != nil {
		return nil, err
	}
	if addGlobalOutput {
		prepare.TestDefinition.AddStdOutput(true)
	}
	if addGlobalInput {
		prepare.TestDefinition.AddInputArtifacts(testdefinition.GetStdInputArtifacts()...)
	}

	return prepare, nil
}

// AddLocation adds a testdef-location to the cloned repos and output artifacts.
func (p *Definition) AddLocation(loc testdefinition.Location) {
	if _, ok := p.config.Repositories[loc.Name()]; ok {
		return
	}
	if loc.Type() != tmv1beta1.LocationTypeGit {
		return
	}
	gitLoc := loc.(*location.GitLocation)
	p.config.Repositories[loc.Name()] = &Repository{Name: loc.Name(), URL: gitLoc.Info.Repo, Revision: gitLoc.Info.Revision}

	p.TestDefinition.AddOutputArtifacts(argov1.Artifact{
		Name:       loc.Name(),
		GlobalName: loc.Name(),
		Path:       fmt.Sprintf("%s/%s", testmachinery.TM_REPO_PATH, loc.Name()),
	})
}

// AddRepositoriesAsArtifacts adds all git config to be cloned as json array to the prepare step.
func (p *Definition) AddRepositoriesAsArtifacts() error {
	repoJSON, err := json.Marshal(p.config)
	if err != nil {
		return fmt.Errorf("cannot add config to prepare step: %s", err.Error())
	}
	p.TestDefinition.AddInputArtifacts(argov1.Artifact{
		Name: "repos",
		Path: PrepareConfigPath,
		ArtifactLocation: argov1.ArtifactLocation{
			Raw: &argov1.RawArtifact{
				Data: string(repoJSON),
			},
		},
	})

	return nil
}

func (p *Definition) addNetrcFile() error {
	netrc := ""

	for _, secret := range testmachinery.GetGitHubSecrets() {
		u, err := url.Parse(secret.HttpUrl)
		if err != nil {
			return errors.Wrapf(err, "%s is not a valid URL", secret.HttpUrl)
		}
		netrc = netrc + fmt.Sprintf("machine %s\nlogin %s\npassword %s\n\n", u.Hostname(), secret.TechnicalUser.Username, secret.TechnicalUser.AuthToken)
	}

	p.TestDefinition.AddInputArtifacts(argov1.Artifact{
		Name: "netrc",
		Path: "/root/.netrc",
		ArtifactLocation: argov1.ArtifactLocation{
			Raw: &argov1.RawArtifact{
				Data: netrc,
			},
		},
	})
	return nil
}
