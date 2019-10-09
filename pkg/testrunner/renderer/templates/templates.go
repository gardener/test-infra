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

package templates

import (
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/hostscheduler"
)

var TestInfraLocationName = "tm"
var DefaultLocationSetName = "default"

var TestInfraLocation = v1beta1.LocationSet{
	Name:    TestInfraLocationName,
	Default: false,
	Locations: []v1beta1.TestLocation{
		{
			Type:     v1beta1.LocationTypeGit,
			Repo:     common.TestInfraRepo,
			Revision: "master",
		},
	},
}

func GetDefaultLocationsSet(cfg GardenerConfig) v1beta1.LocationSet {
	set := v1beta1.LocationSet{
		Name:      DefaultLocationSetName,
		Default:   true,
		Locations: []v1beta1.TestLocation{},
	}

	if cfg.Version != "" {
		set.Locations = []v1beta1.TestLocation{
			{
				Type:     v1beta1.LocationTypeGit,
				Repo:     common.GardenerRepo,
				Revision: cfg.Version,
			},
		}
	}
	if cfg.Commit != "" {
		set.Locations = []v1beta1.TestLocation{
			{
				Type:     v1beta1.LocationTypeGit,
				Repo:     common.GardenerRepo,
				Revision: cfg.Commit,
			},
		}
	}

	return set
}

// GetGardenSetupLocation returns the location set for test machinery steps like lock and release the hostcluster, or fetching of logs, etc.
func GetGardenSetupLocation(name, revision string) v1beta1.LocationSet {
	return v1beta1.LocationSet{
		Name:    name,
		Default: false,
		Locations: []v1beta1.TestLocation{
			{
				Type:     v1beta1.LocationTypeGit,
				Repo:     common.GardenSetupRepo,
				Revision: revision,
			},
		},
	}
}

// GetExitTestFlow returns the default exit handler for gardener tests.
// The testflow consists of the following flow
// - fetch logs from all gardener components
// - delete all shoots that may be left in the gardener
// - delete gardener
// - cleanup and release the host cluster
func GetExitTestFlow(hostprovider hostscheduler.Provider, gsLocationSet string, createGardenerStep *v1beta1.DAGStep) v1beta1.TestFlow {
	deleteGardener := GetStepDeleteGardener(createGardenerStep, gsLocationSet, []string{"clean-gardener"}, false)
	deleteGardener.ArtifactsFrom = ""
	deleteGardener.UseGlobalArtifacts = true
	deleteGardener.Definition.Condition = v1beta1.ConditionTypeError

	releaseHostCluster := GetStepReleaseHost(hostprovider, []string{deleteGardener.Name}, true)
	releaseHostCluster.UseGlobalArtifacts = true
	releaseHostCluster.Definition.Condition = v1beta1.ConditionTypeError
	return v1beta1.TestFlow{
		{
			Name: "fetch-logs",
			Definition: v1beta1.StepDefinition{
				Name:            "log-gardener",
				Condition:       v1beta1.ConditionTypeError,
				ContinueOnError: true,
				LocationSet:     &TestInfraLocationName,
			},
			UseGlobalArtifacts: true,
		},
		{
			Name: "clean-gardener",
			Definition: v1beta1.StepDefinition{
				Name:            "clean-gardener",
				Condition:       v1beta1.ConditionTypeError,
				ContinueOnError: true,
				LocationSet:     &TestInfraLocationName,
			},
			UseGlobalArtifacts: true,
			DependsOn:          []string{"fetch-logs"},
		},
		&deleteGardener,
		&releaseHostCluster,
	}
}
