// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	intconfig "github.com/gardener/test-infra/pkg/apis/config"
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
	tmClient   client.Client

	tmConfig *intconfig.Configuration
	State    OperationState
}

type OperationState struct {
	Objects []client.Object
}
