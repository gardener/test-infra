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

package _default

import (
	"fmt"
	"github.com/Masterminds/semver"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/gardener/test-infra/pkg/hostscheduler/gkescheduler"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/testrunner/renderer"
	"github.com/gardener/test-infra/pkg/testrunner/renderer/templates"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// Config is used to render a default gardener test
type Config struct {
	// Namespace of the testrun
	Namespace string

	// Provider where the host clusters are selected from
	HostProvider hostscheduler.Provider

	// CloudProvider of the base cluster (has to be specified to install the correct credentials and cloudprofiles for the soil/seeds)
	BaseClusterCloudprovider gardenv1beta1.CloudProvider

	// Revision for the gardensetup repo that i sused to install gardener
	GardenSetupRevision string

	// List of components (by default read from a component_descriptor) that are added as locations
	Components componentdescriptor.ComponentList

	// Gardener specific configuration
	Gardener templates.GardenerConfig

	// Gardener tests that do not depend on shoots and run after the shoot tests
	Tests renderer.TestsFunc

	// Shoot test flavor configuration
	Shoots ShootsConfig
}

// ShootsConfig describes the flavors of the shoots that are created by the test.
// The resulting shoot test matrix consists of
// - shoot tests for all specified cloudproviders with all specified kubernets with the default test
// - shoot tests for all specified cloudproviders for all specified tests
type ShootsConfig struct {
	// Shoot/Project namespace where the shoots are created
	Namespace string

	// Default test that is used for all cloudprovider and kubernetes flavors.
	DefaultTest renderer.TestsFunc

	// Specific tests that get their own shoot per cloudprovider and run in parallel to the default tests
	Tests []renderer.TestsFunc

	// Kubernetes versions to test
	KubernetesVersions []string

	// Cloiudproviders to test
	CloudProviders []gardenv1beta1.CloudProvider
}

// Render renders a gardener default test which consists of
// - lock host
// - create gardener (with garden-setup)
// - create shoots in different flavors (cloudprovider, k8s versions)
// - test shoots with different tests that can be specified by a tests function
// - delete shoots
// - delete gardener
// - release host
func Render(cfg *Config) (*v1beta1.Testrun, error) {
	if cfg.HostProvider == gkescheduler.Name {
		cfg.BaseClusterCloudprovider = gardenv1beta1.CloudProviderGCP
	}

	shoots := make([]*shoot, 0)
	for _, cp := range cfg.Shoots.CloudProviders {
		for _, v := range cfg.Shoots.KubernetesVersions {
			version, err := semver.NewVersion(v)
			if err != nil {
				return nil, err
			}
			shoots = append(shoots, &shoot{
				Type:      cp,
				Suffix:    fmt.Sprintf("%s-%d", cp, version.Minor()),
				TestsFunc: cfg.Shoots.DefaultTest,
				Config: &templates.CreateShootConfig{
					ShootName:  fmt.Sprintf("%s-%s", cp, util.RandomString(3)),
					Namespace:  cfg.Shoots.Namespace,
					K8sVersion: v,
				},
			})
		}
		for _, test := range cfg.Shoots.Tests {
			shoots = append(shoots, &shoot{
				Type:      cp,
				Suffix:    fmt.Sprintf("%s-%s", cp, util.RandomString(3)),
				TestsFunc: test,
				Config: &templates.CreateShootConfig{
					ShootName:  fmt.Sprintf("%s-%s", cp, util.RandomString(3)),
					Namespace:  cfg.Shoots.Namespace,
					K8sVersion: cfg.Shoots.KubernetesVersions[0],
				},
			})
		}
	}

	tr, err := testrun(cfg, shoots)
	if err != nil {
		return nil, err
	}

	if err := renderer.AddBOMLocationsToTestrun(tr, "default", cfg.Components, true); err != nil {
		return nil, err
	}
	return tr, nil
}

// Validate validates the testrun render config for the default template
func Validate(cfg *Config) error {
	if cfg == nil {
		return errors.New("config needs to be defined")
	}

	var result *multierror.Error
	if cfg.HostProvider == "" {
		result = multierror.Append(result, errors.New("a host provider needs to be provided"))
	}
	if cfg.BaseClusterCloudprovider == "" {
		result = multierror.Append(result, errors.New("the cloudprovider of the hostcluster needs to be defined"))
	}
	if cfg.Shoots.DefaultTest == nil {
		result = multierror.Append(result, errors.New("a default test needs to be defined"))
	}
	if len(cfg.Shoots.KubernetesVersions) == 0 {
		result = multierror.Append(result, errors.New("at least one kubernetes version has to be defined"))
	}
	if cfg.Shoots.Namespace == "" {
		result = multierror.Append(result, errors.New("the shoot project namespace has to be defined"))
	}

	return util.ReturnMultiError(result)
}
