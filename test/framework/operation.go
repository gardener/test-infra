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
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/apis/config"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/test/utils"
)

// Client returns the kubernetes client of the current test cluster
func (o *Operation) Client() client.Client {
	return o.tmClient
}

// GetKubeconfigPath returns the path to the current kubeconfig
func (o *Operation) GetKubeconfigPath() string {
	return o.testConfig.TmKubeconfigPath
}

// TestMachineryNamespace returns the current namespace where the testmachinery components are running.
func (o *Operation) TestMachineryNamespace() string {
	return o.testConfig.TmNamespace
}

// TestNamespace returns the name of the current test namespace
func (o *Operation) TestNamespace() string {
	return o.testConfig.Namespace
}

// Commit returns the current commit sha of the test-infra repo
func (o *Operation) Commit() string {
	return o.testConfig.CommitSha
}

func (o *Operation) S3Endpoint() string {
	return o.testConfig.S3Endpoint
}

// Log returns the test logger
func (o *Operation) Log() logr.Logger {
	return o.log
}

// IsLocal indicates if the test is running against a local testmachinery controller
func (o *Operation) IsLocal() bool {
	return o.testConfig.Local
}

// S3Config returns the s3 testConfig that is used by the testmachinery to test
func (o *Operation) S3Config() (*config.S3, error) {
	if len(o.testConfig.S3Endpoint) == 0 {
		return nil, errors.New("no s3 endpoint is defined")
	}
	if o.tmConfig.S3 == nil {
		return nil, errors.New("no s3 config is defined")
	}
	return &config.S3{
		Server: config.S3Server{
			Endpoint: o.testConfig.S3Endpoint,
		},
		AccessKey:  o.tmConfig.S3.AccessKey,
		SecretKey:  o.tmConfig.S3.SecretKey,
		BucketName: o.tmConfig.S3.BucketName,
	}, nil
}

// WaitForClusterReadiness waits until all Test Machinery components are ready
func (o *Operation) WaitForClusterReadiness(maxWaitTime time.Duration) error {
	if o.IsLocal() {
		return nil
	}
	return utils.WaitForClusterReadiness(o.Log(), o.tmClient, o.testConfig.TmNamespace, maxWaitTime)
}

func (o *Operation) RunTestrunUntilCompleted(ctx context.Context, tr *tmv1beta1.Testrun, phase argov1.WorkflowPhase, timeout time.Duration) (*tmv1beta1.Testrun, *argov1.Workflow, error) {
	return utils.RunTestrunUntilCompleted(ctx, o.Log().WithValues("namespace", o.TestNamespace()), o.Client(), tr, phase, timeout)
}

func (o *Operation) RunTestrun(ctx context.Context, tr *tmv1beta1.Testrun, phase argov1.WorkflowPhase, timeout time.Duration, watchFunc utils.WatchFunc) (*tmv1beta1.Testrun, *argov1.Workflow, error) {
	return utils.RunTestrun(ctx, o.Log().WithValues("namespace", o.TestNamespace()), o.Client(), tr, phase, timeout, watchFunc)
}

// AppendObject adds a kubernetes objects to the start of the state's objects.
// These objects are meant to be cleaned up after the test has run.
func (s *OperationState) AppendObject(obj client.Object) {
	if s.Objects == nil {
		s.Objects = make([]client.Object, 0)
	}
	s.Objects = append([]client.Object{obj}, s.Objects...)
}
