// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package garbagecollection_test

import (
	"testing"
	"time"

	"github.com/minio/minio-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/test-infra/test/framework"
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
	cfg = framework.RegisterFlags(nil)
}

var _ = BeforeSuite(func() {
	var err error

	operation, err = framework.New(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)), cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(operation.WaitForClusterReadiness(ClusterReadinessTimeout)).ToNot(HaveOccurred())

}, InitializationTimeout.Seconds())

var _ = AfterSuite(func() {
	operation.AfterSuite()
}, CleanupTimeout.Seconds())
