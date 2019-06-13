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
	"github.com/gardener/test-infra/pkg/testmachinery/locations/location"
	"net/url"
	"path"

	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/testmachinery/testdefinition"
	"github.com/gardener/test-infra/pkg/util/strconf"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/util"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
)

// New creates the TM prepare step
// The step clones all needed github repositories and outputs these repos as argo artifacts with the name "repoOwner-repoName-revision".
func New(name string, addGlobalInput bool) (*Definition, error) {
	td := &tmv1beta1.TestDefinition{
		Metadata: tmv1beta1.TestDefMetadata{
			Name: name,
		},
	}
	template := &argov1.Template{
		Name: fmt.Sprintf("%s-%s", name, util.RandomString(5)),
		Metadata: argov1.Metadata{
			Annotations: map[string]string{
				"testmachinery.sapcloud.io/TestDefinition": "Prepare",
			},
		},
		ActiveDeadlineSeconds: &testdefinition.DefaultActiveDeadlineSeconds,
		Container: &apiv1.Container{
			Image:   testmachinery.PREPARE_IMAGE,
			Command: []string{"/tm/prepare", "/tm/repos.json"},
			Env: []apiv1.EnvVar{
				{
					Name:  "TM_KUBECONFIG_PATH",
					Value: testmachinery.TM_KUBECONFIG_PATH,
				},
				{
					Name:  "TM_PHASE",
					Value: "{{inputs.parameters.phase}}",
				},
				{
					Name:  "TM_REPO_PATH",
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
	prepare := &Definition{&testdefinition.TestDefinition{Info: td, Template: template}, addGlobalInput, []*Repository{}}

	if err := prepare.addNetrcFile(); err != nil {
		return nil, err
	}
	if addGlobalInput {
		prepare.TestDefinition.AddInputArtifacts(testdefinition.GetStdInputArtifacts()...)
	}

	return prepare, nil
}

// AddLocation adds a testdef-location to the cloned repos and output artifacts.
func (p *Definition) AddLocation(loc testdefinition.Location) {
	if loc.Type() == tmv1beta1.LocationTypeGit {
		gitLoc := loc.(*location.GitLocation)
		p.repositories = append(p.repositories, &Repository{Name: loc.Name(), URL: gitLoc.Info.Repo, Revision: gitLoc.Info.Revision})

		p.TestDefinition.AddOutputArtifacts(argov1.Artifact{
			Name:       loc.Name(),
			GlobalName: loc.Name(),
			Path:       fmt.Sprintf("%s/%s", testmachinery.TM_REPO_PATH, loc.Name()),
		})
	}
}

// AddRepositoriesAsArtifacts adds all git repositories to be cloned as json array to the prepare step.
func (p *Definition) AddRepositoriesAsArtifacts() error {
	repoJSON, err := json.Marshal(p.repositories)
	if err != nil {
		return fmt.Errorf("Cannot add repositories to prepare step: %s", err.Error())
	}
	p.TestDefinition.AddInputArtifacts(argov1.Artifact{
		Name: "repos",
		Path: "/tm/repos.json",
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
			log.Error("Cannot parse shoot config")
			return err
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
		})
		p.TestDefinition.AddVolumeMount(cfg.Name(), kubeconfigPath, path.Base(kubeconfigPath), true)
		return p.TestDefinition.AddVolumeFromConfig(cfg)
	}
	return fmt.Errorf("Undefined StringSecType %s", string(kubeconfig.Type))
}

func (p *Definition) addNetrcFile() error {
	netrc := ""

	for _, secret := range testmachinery.GetConfig().GitSecrets {
		u, err := url.Parse(secret.HttpUrl)
		if err != nil {
			log.Debugf("%s is not a valid URL: %s", secret.HttpUrl, err.Error())
			continue
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
