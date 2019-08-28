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

package garbagecollection_test

import (
	"github.com/gardener/test-infra/test/framework"
	"github.com/minio/minio-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"

	"testing"
)

func TestGarbageCollection(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Garbage collection Integration Test Suite")
}

const (
	InitializationTimeout        = 20 * time.Minute
	ClusterReadinessTimeout      = 10 * time.Minute
	MinioServiceReadinessTimeout = 5 * time.Minute
	CleanupTimeout               = 1 * time.Minute

	TestrunDurationTimeout = 10 * time.Minute
)

var (
	cfg         *framework.Config
	operation   *framework.Operation
	minioClient *minio.Client
	minioBucket string
)

func init() {
	cfg = framework.InitFlags(nil)
}

var _ = BeforeSuite(func() {
	var err error

	operation, err = framework.New(zap.LoggerTo(GinkgoWriter, true), cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(operation.WaitForClusterReadiness(ClusterReadinessTimeout)).ToNot(HaveOccurred())

	osConfig, err := operation.WaitForMinioServiceReadiness(MinioServiceReadinessTimeout)
	Expect(err).ToNot(HaveOccurred())

	minioBucket = osConfig.BucketName
	minioClient, err = minio.New(osConfig.Endpoint, osConfig.AccessKey, osConfig.SecretKey, false)
	Expect(err).ToNot(HaveOccurred())
}, InitializationTimeout.Seconds())

var _ = AfterSuite(func() {
	operation.AfterSuite()
}, CleanupTimeout.Seconds())
