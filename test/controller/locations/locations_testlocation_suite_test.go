// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package locations_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var _ = Describe("Locations TestLocations tests", func() {

	It("should run a test with a TestLocations", func() {
		ctx := context.Background()
		defer ctx.Done()
		tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
		tr.Spec.LocationSets = nil
		tr.Spec.TestLocations = []tmv1beta1.TestLocation{
			{
				Type:     tmv1beta1.LocationTypeGit,
				Repo:     "https://github.com/gardener/test-infra.git",
				Revision: operation.Commit(),
			},
		}

		tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, tmv1beta1.RunPhaseSuccess, TestrunDurationTimeout)
		defer utils.DeleteTestrun(operation.Client(), tr)
		Expect(err).ToNot(HaveOccurred())
	})
})
