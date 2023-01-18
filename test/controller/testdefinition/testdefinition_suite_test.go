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

package testdefinition_test

import (
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/test-infra/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTestDefinitions(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testrun testdefinition Integration Test Suite")
}

const (
	InitializationTimeout = 10 * time.Minute
	CleanupTimeout        = 1 * time.Minute

	TestrunDurationTimeout = 6 * time.Minute
)

var (
	cfg       *framework.Config
	operation *framework.Operation
)

func init() {
	cfg = framework.RegisterFlags(nil)
}

var _ = BeforeSuite(func() {
	var err error
	operation, err = framework.New(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)), cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(operation.WaitForClusterReadiness(InitializationTimeout)).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	operation.AfterSuite()
}, CleanupTimeout.Seconds())
