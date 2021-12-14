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

import "time"

// TestMachinery holds information about the testmachinery
type TestMachinery struct {
	// Namespace is the namespace the testmachinery is deployed to.
	Namespace string `json:"namespace,omitempty"`

	// TestDefPath is the repository path where the Test Machinery should search for testdefinitions.
	TestDefPath string `json:"testdefPath"`

	// Locations contains all testlocation configurations
	Locations Locations `json:"locations,omitempty"`

	// PrepareImage is the prepare image that is used in the prepare and postprepare step.
	PrepareImage string `json:"prepareImage"`

	// PrepareImage is the base image that is used as the default image if a TestDefinition does not define an image.
	BaseImage string `json:"baseImage"`

	// Local indicates if the controller is run locally.
	Local bool `json:"local,omitempty"`

	// Insecure indicates that the testmachinery runs insecure.
	Insecure bool `json:"insecure,omitempty"`

	// RetryTimeout defines the timeout when a testrun is not retried anymore and goes into failed state.
	// The string is expected to be passed as golang duration parsable format.
	RetryTimeout string `json:"retryTimeout,omitempty"`

	// RetryTimeoutDuration is the parsed value of the RetryTimeout.
	RetryTimeoutDuration *time.Duration `json:"-"`

	// DisableCollector disables the collection of test results and their ingestion into elasticsearch.
	DisableCollector bool `json:"disableCollector"`

	// CleanWorkflowPods indicates if workflow pods should be directly cleaned up by the testmachinery.
	CleanWorkflowPods bool `json:"cleanWorkflowPods,omitempty"`

	//LandscapeMappings defines how to connect to landscapes using the respective OpenIDConnect IDP
	LandscapeMappings []LandscapeMapping `json:"landscapeMappings,omitempty"`
}

// Locations defines test location configurations.
type Locations struct {
	// ExcludeDomains is a list of domains that should be excluded and no test definition fetched from.
	// Note that the domain and all its subdomains are ignored.
	ExcludeDomains []string `json:"excludeDomains,omitempty"`
}

// GitHub holds all github related information needed in the testmachinery.
type GitHub struct {
	Cache *GitHubCache `json:"cache,omitempty"`

	// SecretsPath is the path to the github secrets file
	SecretsPath string `json:"secretsPath,omitempty"`
}

// GitHubCache is the github cache configuration
type GitHubCache struct {
	CacheDir        string `json:"cacheDir,omitempty"`
	CacheDiskSizeGB int    `json:"cacheDiskSizeGB,omitempty"`
	MaxAgeSeconds   int    `json:"maxAgeSeconds,omitempty"`
}

//LandscapeMapping defines how to connect to a landscape using an OpenIDConnect IDP
type LandscapeMapping struct {
	// Namespace indicates the namespace where TestRuns for a specific landscape should run
	Namespace string `json:"namespace,omitempty"`

	// ApiServerUrl discloses the allowed target APi server
	ApiServerUrl string `json:"apiServerUrl,omitempty"`

	// Audience is the audience/client ID, which has to be used for requests to this landscape
	Audience string `json:"audience"`

	//ExpirationSeconds specifies the lifetime of issued tokens
	ExpirationSeconds int64 `json:"expirationSeconds"`

	//AllowUntrustedUsage defines, if a token is allowed to be used in untrusted steps
	AllowUntrustedUsage bool `json:"allowUntrustedUsage"`
}
