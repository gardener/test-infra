// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package locations_test

import (
	"context"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var _ = Describe("Locations LocationSets tests", func() {

	Context("LocationSets", func() {
		It("should run a test with one location set and a specific default location", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

		})

		It("should use the first location set as default", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.LocationSets = []tmv1beta1.LocationSet{
				{
					Name: "default",
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra.git",
							Revision: operation.Commit(),
						},
					},
				},
				{
					Name: "non-default",
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra-non.git",
							Revision: "master",
						},
					},
				},
			}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should use the second location set as default", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.LocationSets = []tmv1beta1.LocationSet{
				{
					Name: "non-default",
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra-non.git",
							Revision: "master",
						},
					},
				},
				{
					Name:    "default",
					Default: true,
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra.git",
							Revision: operation.Commit(),
						},
					},
				},
			}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("flow step", func() {
		It("should use a specific location set", func() {
			ctx := context.Background()
			defer ctx.Done()

			setName := "default"
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.LocationSets = []tmv1beta1.LocationSet{
				{
					Name: "non-default",
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra-non.git",
							Revision: "master",
						},
					},
				},
				{
					Name: setName,
					Locations: []tmv1beta1.TestLocation{
						{
							Type:     tmv1beta1.LocationTypeGit,
							Repo:     "https://github.com/gardener/test-infra.git",
							Revision: operation.Commit(),
						},
					},
				},
			}
			tr.Spec.TestFlow[0].Definition.LocationSet = &setName

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})
