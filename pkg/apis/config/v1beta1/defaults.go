// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package v1beta1

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/version"
	"k8s.io/apimachinery/pkg/runtime"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_ControllerConfig sets default values for the ControllerConfig objects
func SetDefaults_ControllerConfig(obj *ControllerConfig) {
	if obj.MaxConcurrentSyncs == 0 {
		obj.MaxConcurrentSyncs = 1
	}

	if len(obj.HealthAddr) == 0 {
		obj.HealthAddr = ":8081"
	}

	if len(obj.MetricsAddr) == 0 {
		obj.MetricsAddr = ":8080"
	}

	if obj.WebhookConfig.Port == 0 {
		obj.WebhookConfig.Port = 443
	}
}

// SetDefaults_TestMachineryConfiguration sets default values for the TestMachineryConfiguration objects
func SetDefaults_TestMachineryConfiguration(obj *TestMachineryConfiguration) {
	if len(obj.TestDefPath) == 0 {
		obj.TestDefPath = ".test-defs"
	}
	if len(obj.PrepareImage) == 0 {
		obj.PrepareImage = fmt.Sprintf("eu.gcr.io/gardener-project/gardener/testmachinery/prepare-step:%s", version.Get().GitVersion)
	}
	if len(obj.BaseImage) == 0 {
		obj.BaseImage = fmt.Sprintf("eu.gcr.io/gardener-project/gardener/testmachinery/base-step:%s", version.Get().GitVersion)
	}

	if len(obj.Namespace) == 0 {
		obj.Namespace = "default"
	}
}
