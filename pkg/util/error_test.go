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
