// SPDX-FileCopyrightText: Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package result

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTestrunnerResult(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testrunner Result Test Suite")
}
