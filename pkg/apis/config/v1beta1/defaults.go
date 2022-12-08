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
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/test-infra/pkg/version"
)

func addDefaultingFuncs(scheme *runtime.Scheme) error {
	return RegisterDefaults(scheme)
}

// SetDefaults_Configuration sets default values for the Configuration objects
func SetDefaults_Configuration(obj *Configuration) {
	SetDefaults_ControllerConfig(&obj.Controller)
	SetDefaults_TestMachineryConfiguration(&obj.TestMachinery)
}

// SetDefaults_ControllerConfig sets default values for the Controller objects
func SetDefaults_ControllerConfig(obj *Controller) {
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

	if len(obj.DependencyHealthCheck.Namespace) == 0 {
		obj.DependencyHealthCheck.Namespace = "default"
	}

	if len(obj.DependencyHealthCheck.DeploymentName) == 0 {
		obj.DependencyHealthCheck.DeploymentName = "workflow-controller"
	}

	if obj.DependencyHealthCheck.Interval.Duration == 0 {
		obj.DependencyHealthCheck.Interval.Duration = time.Minute
	}
}

// SetDefaults_TestMachineryConfiguration sets default values for the TestMachinery objects
func SetDefaults_TestMachineryConfiguration(obj *TestMachinery) {
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

// SetDefaults_Webserver sets default values for the Webserver objects
func SetDefaults_Webserver(obj *Webserver) {
	if obj.HTTPPort == 0 {
		obj.HTTPPort = 80
	}
	if obj.HTTPSPort == 0 {
		obj.HTTPSPort = 443
	}
}

// SetDefaults_GitHubBot sets default values for the GitHubBot objects
func SetDefaults_GitHubBot(obj *GitHubBot) {
	if len(obj.ApiUrl) == 0 {
		obj.ApiUrl = "https://api.github.com"
	}
	if len(obj.ConfigurationFilePath) == 0 {
		obj.ConfigurationFilePath = ".ci/tm-config.yaml"
	}
}

// SetDefaults_Dashboard sets default values for the Dashboard objects
func SetDefaults_Dashboard(obj *Dashboard) {
	if len(obj.UIBasePath) == 0 {
		obj.UIBasePath = "/app"
	}

	if obj.Authentication.GitHub != nil {
		SetDefaults_GitHubAuthentication(obj.Authentication.GitHub)
	}
}

// SetDefaults_GitHubAuthentication sets default values for the GitHubAuthentication objects
func SetDefaults_GitHubAuthentication(obj *GitHubAuthentication) {
	if len(obj.Organization) == 0 {
		obj.Organization = "gardener"
	}
	if len(obj.Hostname) == 0 {
		obj.Hostname = "github.com"
	}
}
