// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package locations_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/test-infra/test/framework"
)

func TestLocations(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testrun locations Integration Test Suite")
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
