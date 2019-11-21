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

package prepare

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery/locations/location"
	"github.com/pkg/errors"
	"net/url"
	"path"

	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/util/strconf"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/util"
	apiv1 "k8s.io/api/core/v1"
)

// New creates the TM prepare step
// The step clones all needed github config and outputs these repos as argo artifacts with the name "repoOwner-repoName-revision".
func New(name string, addGlobalInput, addGlobalOutput bool) (*Definition, error) {
	td := testdefinition.NewEmpty()
	td.Info = &tmv1beta1.TestDefinition{
		Metadata: tmv1beta1.TestDefMetadata{
			Name: name,
		},
	}
	td.Template = &argov1.Template{
		Name: fmt.Sprintf("%s-%s", name, util.RandomString(5)),
		Metadata: argov1.Metadata{
			Annotations: map[string]string{
				"testmachinery.sapcloud.io/TestDefinition": "Prepare",
				common.AnnotationSystemStep:                "true",
			},
		},
		ActiveDeadlineSeconds: &testdefinition.DefaultActiveDeadlineSeconds,
		Container: &apiv1.Container{
			Image:   testmachinery.PREPARE_IMAGE,
			Command: []string{"/tm/prepare", PrepareConfigPath},
			Env: []apiv1.EnvVar{
				{
					Name:  testmachinery.TM_PHASE_NAME,
					Value: "{{inputs.parameters.phase}}",
				},
				{
					Name:  testmachinery.TM_REPO_PATH_NAME,
					Value: testmachinery.TM_REPO_PATH,
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
		Repositories: make(map[string]*Repository, 0),
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

// AddKubeconfig adds all defined kubeconfigs as files to the prepare pod
func (p *Definition) AddKubeconfigs(kubeconfigs tmv1beta1.TestrunKubeconfigs) error {
	if kubeconfigs.Gardener != nil {
		if err := p.addKubeconfig("gardener", kubeconfigs.Gardener); err != nil {
			return err
		}
	}
	if kubeconfigs.Seed != nil {
		if err := p.addKubeconfig("seed", kubeconfigs.Seed); err != nil {
			return err
		}
	}
	if kubeconfigs.Shoot != nil {
		if err := p.addKubeconfig("shoot", kubeconfigs.Shoot); err != nil {
			return err
		}
	}
	return nil
}

func (p *Definition) addKubeconfig(name string, kubeconfig *strconf.StringOrConfig) error {
	kubeconfigPath := fmt.Sprintf("%s/%s.config", testmachinery.TM_KUBECONFIG_PATH, name)
	if kubeconfig.Type == strconf.String {
		kubeconfig, err := base64.StdEncoding.DecodeString(kubeconfig.String())
		if err != nil {
			return errors.Wrapf(err, "unable to parse kubeconfig")
		}
		p.TestDefinition.AddInputArtifacts(argov1.Artifact{
			Name: name,
			Path: kubeconfigPath,
			ArtifactLocation: argov1.ArtifactLocation{
				Raw: &argov1.RawArtifact{
					Data: string(kubeconfig),
				},
			},
		})
		return nil
	}
	if kubeconfig.Type == strconf.Config {
		cfg := config.NewElement(&tmv1beta1.ConfigElement{
			Type:      tmv1beta1.ConfigTypeFile,
			Name:      name,
			Path:      kubeconfigPath,
			ValueFrom: kubeconfig.Config(),
		}, config.LevelTestDefinition)
		p.TestDefinition.AddVolumeMount(cfg.Name(), kubeconfigPath, path.Base(kubeconfigPath), true)
		return p.TestDefinition.AddVolumeFromConfig(cfg)
	}
	return fmt.Errorf("undefined StringSecType %s", string(kubeconfig.Type))
}

func (p *Definition) addNetrcFile() error {
	netrc := ""

	for _, secret := range testmachinery.GetConfig().GitHub.Secrets {
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
