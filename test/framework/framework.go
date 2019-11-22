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
	"context"
	flag "github.com/spf13/pflag"
	"os"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// New creates a new test operation with a logger and a framework configuration.
func New(log logr.Logger, config *Config) (*Operation, error) {

	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	tmClient, err := kubernetes.NewClientFromFile("", config.TmKubeconfigPath, kubernetes.WithClientOptions(client.Options{
		Scheme: testmachinery.TestMachineryScheme,
	}))
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create client from %s", config.TmKubeconfigPath)
	}

	operation := &Operation{
		config:   config,
		log:      log,
		tmClient: tmClient,
	}

	if err := operation.EnsureTestNamespace(context.TODO()); err != nil {
		return nil, err
	}

	return operation, nil
}

// InitFlags adds all framework operation specific flags to the provided flagset.
func InitFlags(flagset *flag.FlagSet) *Config {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	cfg := Config{}

	flagset.StringVar(&cfg.CommitSha, "git-commit-sha", util.Getenv("GIT_COMMIT_SHA", "master"),
		"Commit hash of the current test-infra git")
	flagset.StringVar(&cfg.Namespace, "namespace", util.Getenv("NAMESPACE", ""),
		"testing namespace")
	flagset.StringVar(&cfg.TmKubeconfigPath, "kubeconfig", util.Getenv("TM_KUBECONFIG_PATH", "default"),
		"Kubeconfig path to the testmachinery")
	flagset.StringVar(&cfg.TmNamespace, "tm-namespace", util.Getenv("TM_NAMESPACE", "default"),
		"namespace where the testmachinery is running")
	flagset.StringVar(&cfg.S3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"),
		"s3 endpoint of the s3 storage used by the workflows")

	flagset.BoolVar(&cfg.Local, "local", false,
		"test runs locally which means that prerequisites like readyness of controller and minio is not checked")

	return &cfg
}
