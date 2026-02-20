// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package tests_test

import (
	"github.com/google/go-github/v83/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	ghutil "github.com/gardener/test-infra/pkg/tm-bot/github"
	"github.com/gardener/test-infra/pkg/tm-bot/tests"
)

var _ = Describe("Runs", func() {
	It("should add a Run", func() {
		runs := tests.NewRuns(nil)

		owner := "test"
		repo := "repo"
		event := &ghutil.GenericRequestEvent{
			Number: 0,
			Repository: &github.Repository{
				Name: &repo,
				Owner: &github.User{
					Login: &owner,
				},
			},
		}
		Expect(runs.Add(event, &v1beta1.Testrun{})).NotTo(HaveOccurred())

		Expect(runs.IsRunning(event)).To(BeTrue())
	})

	It("should return the currently running Testrun", func() {
		runs := tests.NewRuns(nil)

		owner := "test"
		repo := "repo"
		event := &ghutil.GenericRequestEvent{
			Number: 0,
			Repository: &github.Repository{
				Name: &repo,
				Owner: &github.User{
					Login: &owner,
				},
			},
		}
		Expect(runs.Add(event, &v1beta1.Testrun{})).NotTo(HaveOccurred())

		run, ok := runs.GetRunning(event)
		Expect(ok).To(BeTrue())
		Expect(run.Testrun).To(Equal(&v1beta1.Testrun{}))
	})

	It("should reject another Run if one is already running", func() {
		runs := tests.NewRuns(nil)

		owner := "test"
		repo := "repo"
		event := &ghutil.GenericRequestEvent{
			Number: 0,
			Repository: &github.Repository{
				Name: &repo,
				Owner: &github.User{
					Login: &owner,
				},
			},
		}
		Expect(runs.Add(event, &v1beta1.Testrun{})).NotTo(HaveOccurred())

		event2 := &ghutil.GenericRequestEvent{
			Number: 0,
			Repository: &github.Repository{
				Name: &repo,
				Owner: &github.User{
					Login: &owner,
				},
			},
		}
		Expect(runs.Add(event2, &v1beta1.Testrun{})).To(HaveOccurred())

		Expect(runs.IsRunning(event)).To(BeTrue())
	})

	It("should add another Run if one is running in a different repo", func() {
		runs := tests.NewRuns(nil)

		owner := "test"
		repo := "repo"
		event := &ghutil.GenericRequestEvent{
			Number: 0,
			Repository: &github.Repository{
				Name: &repo,
				Owner: &github.User{
					Login: &owner,
				},
			},
		}
		Expect(runs.Add(event, &v1beta1.Testrun{})).NotTo(HaveOccurred())

		owner2 := "test2"
		repo2 := "repo2"
		event2 := &ghutil.GenericRequestEvent{
			Number: 0,
			Repository: &github.Repository{
				Name: &repo2,
				Owner: &github.User{
					Login: &owner2,
				},
			},
		}
		Expect(runs.Add(event2, &v1beta1.Testrun{})).NotTo(HaveOccurred())

		Expect(runs.IsRunning(event)).To(BeTrue())
		Expect(runs.IsRunning(event2)).To(BeTrue())
	})
})
