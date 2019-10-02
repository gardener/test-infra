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
	"fmt"
	"strconv"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/hostscheduler"
)

func GetStepLockHost(provider hostscheduler.Provider, cloudprovider gardenv1beta1.CloudProvider) v1beta1.DAGStep {
	return v1beta1.DAGStep{
		Name: "prepare-host",
		Definition: v1beta1.StepDefinition{
			Name:        fmt.Sprintf("tm-scheduler-lock-%s", provider),
			LocationSet: &TestInfraLocationName,
			Config: []v1beta1.ConfigElement{
				{
					Type:  v1beta1.ConfigTypeEnv,
					Name:  "HOST_CLOUDPROVIDER",
					Value: string(cloudprovider),
				},
			},
		},
	}
}

func GetStepReleaseHost(provider hostscheduler.Provider, dependencies []string, clean bool) v1beta1.DAGStep {
	step := v1beta1.DAGStep{
		Name: "release-host",
		Definition: v1beta1.StepDefinition{
			Name:        fmt.Sprintf("tm-scheduler-release-%s", provider),
			LocationSet: &TestInfraLocationName,
			Config: []v1beta1.ConfigElement{
				{
					Type:  v1beta1.ConfigTypeEnv,
					Name:  "CLEAN",
					Value: strconv.FormatBool(clean),
				},
			},
		},
		DependsOn: dependencies,
	}
	return step
}
