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

package testdefinition

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/util"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
)

// NewPrepare creates the TM prepare step
// The step clones all needed github repositories and outputs these repos as argo artifacts with the name "repoOwner-repoName-revision".
func NewPrepare(name string, kubeconfigs tmv1beta1.TestrunKubeconfigs) *PrepareDefinition {
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
		ActiveDeadlineSeconds: &activeDeadlineSeconds,
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
	prepare := &PrepareDefinition{&TestDefinition{Info: td, Template: template}, []*PrepareRepository{}}

	prepare.addNetrcFile()
	prepare.addKubeconfigs(kubeconfigs)
	prepare.TestDefinition.AddSerialStdOutput()

	return prepare
}

// AddLocation adds a tesdeflocation to the cloned repos and output artifacts.
func (p *PrepareDefinition) AddLocation(loc Location) {
	if loc.Type() == tmv1beta1.LocationTypeGit {
		gitLoc := loc.(*GitLocation)
		p.repositories = append(p.repositories, &PrepareRepository{Name: loc.Name(), URL: gitLoc.info.Repo, Revision: gitLoc.info.Revision})

		p.TestDefinition.AddOutputArtifacts(argov1.Artifact{
			Name: loc.Name(),
			Path: fmt.Sprintf("%s/%s", testmachinery.TM_REPO_PATH, loc.Name()),
		})
	}
}

// AddRepositoriesAsArtifacts adds all git repositories to be cloned as json array to the prepare step.
func (p *PrepareDefinition) AddRepositoriesAsArtifacts() error {
	repoJSON, err := json.Marshal(p.repositories)
	if err != nil {
		return fmt.Errorf("Cannot add repositories to prepare step: %s", err.Error())
	}
	p.TestDefinition.Template.Inputs.Artifacts = append(p.TestDefinition.Template.Inputs.Artifacts, argov1.Artifact{
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

// addKubeconfig adds all defined kubeconfigs as files to the prepare pod
func (p *PrepareDefinition) addKubeconfigs(kubeconfigs tmv1beta1.TestrunKubeconfigs) error {

	if kubeconfigs.Gardener != "" {
		kubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigs.Gardener)
		if err != nil {
			log.Error("Cannot parse gardener config")
			return err
		}
		p.TestDefinition.AddInputArtifacts(argov1.Artifact{
			Name: "gardener",
			Path: fmt.Sprintf("%s/gardener.config", testmachinery.TM_KUBECONFIG_PATH),
			ArtifactLocation: argov1.ArtifactLocation{
				Raw: &argov1.RawArtifact{
					Data: string(kubeconfig),
				},
			},
		})
	}
	if kubeconfigs.Seed != "" {
		kubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigs.Seed)
		if err != nil {
			log.Error("Cannot parse seed config")
			return err
		}
		p.TestDefinition.AddInputArtifacts(argov1.Artifact{
			Name: "seed",
			Path: fmt.Sprintf("%s/seed.config", testmachinery.TM_KUBECONFIG_PATH),
			ArtifactLocation: argov1.ArtifactLocation{
				Raw: &argov1.RawArtifact{
					Data: string(kubeconfig),
				},
			},
		})
	}
	if kubeconfigs.Shoot != "" {
		kubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigs.Shoot)
		if err != nil {
			log.Error("Cannot parse shoot config")
			return err
		}
		p.TestDefinition.AddInputArtifacts(argov1.Artifact{
			Name: "shoot",
			Path: fmt.Sprintf("%s/shoot.config", testmachinery.TM_KUBECONFIG_PATH),
			ArtifactLocation: argov1.ArtifactLocation{
				Raw: &argov1.RawArtifact{
					Data: string(kubeconfig),
				},
			},
		})
	}
	return nil
}

func (p *PrepareDefinition) addNetrcFile() error {

	netrc := ""

	for _, secret := range testmachinery.GetConfig().GitSecrets {
		netrc = netrc + fmt.Sprintf("machine %s\nlogin %s\npassword %s\n\n", secret.HttpUrl, secret.TechnicalUser.Username, secret.TechnicalUser.AuthToken)
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
