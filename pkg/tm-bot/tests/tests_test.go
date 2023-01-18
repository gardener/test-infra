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

package tests_test

import (
	"github.com/google/go-github/v49/github"
	. "github.com/onsi/ginkgo"
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
