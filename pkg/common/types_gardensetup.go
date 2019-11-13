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

package common

// GSExtensions defines extensions and their configuration that is consumed by the acre.yaml
type GSExtensions = map[string]GSExtensionConfig

// GSExtensionsConfig defines an extension used in the acre.yaml to specify an Extension
type GSExtensionConfig struct {
	Tag             string `json:"tag,omitempty"`
	Commit          string `json:"commit,omitempty"`
	Branch          string `json:"branch,omitempty"`
	Repository      string `json:"repo,omitemtpy"`
	ImageTag        string `json:"image_tag,omitemtpy"`
	ImageRepository string `json:"image_repo,omitemtpy"`
	ChartPath       string `json:"chart_path,omitemtpy"`
}

// GSDependencies represents the dependency vector of all elements in garden setup
type GSDependencies struct {
	Versions GSVersions `json:"versions"`
}

// GSVersions specifies all elements and their versions
type GSVersions struct {
	Gardener GSGardenerDependency `json:"gardener"`
}

// GSGardenerDependency represents the gardener version and its dependencies in the dependency vector
type GSGardenerDependency struct {
	Core       GSVersion            `json:"core"`
	Extensions map[string]GSVersion `json:"extensions"`
}

// GSVersion is one specific version in the garden setup dependency vector
type GSVersion struct {
	Repository string `json:"repo"`
	Version    string `json:"version"`
}
