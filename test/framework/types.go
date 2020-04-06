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

package framework

import (
	"github.com/gardener/gardener/pkg/client/kubernetes"
	intconfig "github.com/gardener/test-infra/pkg/apis/config"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	TestNamespacePrefix = "tm-it"
)

var (
	CoreSecrets = []string{
		"s3-secret",
	}
)

// Framework operation configuration
type Config struct {
	CommitSha        string
	Namespace        string
	TmKubeconfigPath string
	TmNamespace      string
	S3Endpoint       string

	Local        bool
	TMConfigPath string
}

// Operation is a common set of configuration and functions for running testmachinery integration tests.
type Operation struct {
	testConfig *Config
	log        logr.Logger
	tmClient   kubernetes.Interface

	tmConfig *intconfig.Configuration
	State    OperationState
}

type OperationState struct {
	Objects []runtime.Object
}
