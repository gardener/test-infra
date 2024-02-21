// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validationwebhook_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/test-infra/test/framework"
)

func TestValidationWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation webhook Integration Test Suite")
}

const (
	InitializationTimeout = 10 * time.Minute
	CleanupTimeout        = 1 * time.Minute
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
	operation, err = framework.New(zap.New(zap.UseDevMode(true), zap.WriteTo(GinkgoWriter)), cfg)
	Expect(err).ToNot(HaveOccurred())
	Expect(operation.WaitForClusterReadiness(InitializationTimeout)).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	operation.AfterSuite()
}, CleanupTimeout.Seconds())
