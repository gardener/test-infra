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
	"flag"
	"os"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/util"
	kutil "github.com/gardener/test-infra/pkg/util/kubernetes"
)

// New creates a new test operation with a logger and a framework configuration.
func New(log logr.Logger, config *Config) (*Operation, error) {
	flag.Parse()
	ctx := context.Background()
	defer ctx.Done()

	if err := ValidateConfig(config); err != nil {
		return nil, err
	}

	tmClient, err := kutil.NewClientFromFile(config.TmKubeconfigPath, client.Options{
		Scheme: testmachinery.TestMachineryScheme,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create client from %s", config.TmKubeconfigPath)
	}

	operation := &Operation{
		testConfig: config,
		log:        log,
		tmClient:   tmClient,
	}

	if err := operation.setTestMachineryConfig(ctx); err != nil {
		return nil, err
	}

	if err := operation.EnsureTestNamespace(ctx); err != nil {
		return nil, err
	}

	return operation, nil
}

func (o *Operation) setTestMachineryConfig(ctx context.Context) error {
	var (
		data    []byte
		err     error
		decoder = serializer.NewCodecFactory(testmachinery.ConfigScheme).UniversalDecoder()
	)

	if len(o.testConfig.TMConfigPath) != 0 {
		data, err = os.ReadFile(o.testConfig.TMConfigPath)
		if err != nil {
			return err
		}
	} else {
		secret := &corev1.Secret{}
		if err := o.Client().Get(ctx, client.ObjectKey{Name: "tm-configuration", Namespace: o.TestMachineryNamespace()}, secret); err != nil {
			return err
		}
		data = secret.Data["config.yaml"]
	}

	cfg := &config.Configuration{}
	if _, _, err := decoder.Decode(data, nil, cfg); err != nil {
		return err
	}

	o.tmConfig = cfg
	return nil
}

// RegisterFlags adds all framework operation specific flags to the provided flagset.
func RegisterFlags(flagset *flag.FlagSet) *Config {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	cfg := Config{}

	flagset.StringVar(&cfg.CommitSha, "git-commit-sha", util.Getenv("GIT_COMMIT_SHA", "master"),
		"Commit hash of the current test-infra git")
	flagset.StringVar(&cfg.Namespace, "namespace", util.Getenv("NAMESPACE", ""),
		"testing namespace")
	flagset.StringVar(&cfg.TmKubeconfigPath, "kubecfg", util.Getenv("TM_KUBECONFIG_PATH", "default"),
		"Kubeconfig path to the testmachinery")
	flagset.StringVar(&cfg.TmNamespace, "tm-namespace", util.Getenv("TM_NAMESPACE", "default"),
		"namespace where the testmachinery is running")
	flagset.StringVar(&cfg.S3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"),
		"s3 endpoint of the s3 storage used by the workflows")

	flagset.BoolVar(&cfg.Local, "local", false,
		"test runs locally which means that prerequisites like readyness of controller and minio is not checked")
	flagset.StringVar(&cfg.TMConfigPath, "tm-config", "",
		"path to the testmachinery kubeconfig. It will be read from the cluster if not specified")

	return &cfg
}
