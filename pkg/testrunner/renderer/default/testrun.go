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
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner/renderer"
	"github.com/gardener/test-infra/pkg/testrunner/renderer/templates"
	"github.com/gardener/test-infra/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type shoot struct {
	Type      gardenv1beta1.CloudProvider
	Suffix    string
	Config    *templates.CreateShootConfig
	TestsFunc renderer.TestsFunc
}

func testrun(cfg *Config, shoots []*shoot) (*v1beta1.Testrun, error) {
	gsLocationName := "gs"
	prepareHostCluster := templates.GetStepLockHost(cfg.HostProvider, cfg.BaseClusterCloudprovider)
	createGardener, err := templates.GetStepCreateGardener(gsLocationName, []string{prepareHostCluster.Name}, cfg.BaseClusterCloudprovider, cfg.Shoots.Flavors, cfg.Gardener)
	if err != nil {
		return nil, err
	}

	tr := &v1beta1.Testrun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   cfg.Namespace,
			Annotations: cfg.Annotations,
		},

		Spec: v1beta1.TestrunSpec{
			LocationSets: []v1beta1.LocationSet{
				templates.GetDefaultLocationsSet(cfg.Gardener),
				templates.TestInfraLocation,
				templates.GetGardenSetupLocation(gsLocationName, cfg.GardenSetupRevision),
			},
			Config: []v1beta1.ConfigElement{
				templates.GetConfigGardenerPrefix(),
			},
			TestFlow: v1beta1.TestFlow{
				&prepareHostCluster,
				&createGardener,
			},
		},
	}

	deps := make([]string, len(shoots))
	for i, shootConfig := range shoots {
		steps, err := GetShootTest(cfg.Gardener, shootConfig, []string{createGardener.Name})
		if err != nil {
			return nil, err
		}

		tr.Spec.TestFlow = append(tr.Spec.TestFlow, steps...)
		deps[i] = steps[1].Name
	}

	// add gardener tests
	if cfg.Tests != nil {
		var gardenerIntegrationTests []*v1beta1.DAGStep
		gardenerIntegrationTests, deps, err = cfg.Tests(fmt.Sprintf("gardener-it-%s-", util.RandomString(3)), deps)
		if err != nil {
			return nil, err
		}
		tr.Spec.TestFlow = append(tr.Spec.TestFlow, gardenerIntegrationTests...)
	}

	deleteGardener := templates.GetStepDeleteGardener(&createGardener, gsLocationName, deps, cfg.Pause)
	releaseHostCluster := templates.GetStepReleaseHost(cfg.HostProvider, []string{deleteGardener.Name}, false)
	tr.Spec.TestFlow = append(tr.Spec.TestFlow, &deleteGardener, &releaseHostCluster)

	tr.Spec.OnExit = templates.GetExitTestFlow(cfg.HostProvider, gsLocationName, &createGardener)

	return tr, nil
}

// GetShootTest creates one shoot test for a config.
// This test consists of these test steps:
// - create-shoot
// - tests
// - delete-shoot
func GetShootTest(gardenerConfig templates.GardenerConfig, shootConfig *shoot, dependencies []string) ([]*v1beta1.DAGStep, error) {
	createShootStep, createShootStepName, err := templates.GetStepCreateShoot(gardenerConfig, shootConfig.Type, fmt.Sprintf("create-%s", shootConfig.Suffix), dependencies, shootConfig.Config)
	if err != nil {
		return nil, err
	}
	//defaultTestStep := templates.GetTestStepWithLabels(fmt.Sprintf("tests-%s", shootConfig.Suffix), []string{createShootStep.Name}, shootConfig.TestLabel, string(testmachinery.TestLabelShoot))
	tests, testDep, err := shootConfig.TestsFunc(fmt.Sprintf("tests-%s", shootConfig.Suffix), []string{createShootStepName})
	deleteShootStep := templates.GetStepDeleteShoot(fmt.Sprintf("delete-%s", shootConfig.Suffix), createShootStepName, shootConfig.Config.ShootName, testDep)

	return append(append(createShootStep, &deleteShootStep), tests...), nil
}
