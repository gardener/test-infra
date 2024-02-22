// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"github.com/hashicorp/go-multierror"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/util"
)

var _ = Describe("error util", func() {
	It("should return nil if the error is nil ", func() {
		Expect(util.ReturnMultiError(nil)).ToNot(HaveOccurred())
	})
	It("should return nil if the multierr is nil", func() {
		var err *multierror.Error
		Expect(util.ReturnMultiError(err)).ToNot(HaveOccurred())
	})
	It("should return nil if the multierr contains 0 errors", func() {
		var err *multierror.Error
		err = multierror.Append(err, nil)
		Expect(util.ReturnMultiError(err)).ToNot(HaveOccurred())
	})
})
