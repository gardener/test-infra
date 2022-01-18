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

package testrunner_run_test

import (
	"context"
	"time"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testrunner execution tests", func() {

	var (
		testrunConfig testrunner.Config
	)

	BeforeEach(func() {
		testrunConfig = testrunner.Config{
			Namespace: operation.TestNamespace(),
			Timeout:   InitializationTimeout,
		}
	})

	Context("testrun", func() {
		It("should run a single testrun", func() {
			ctx := context.Background()
			defer ctx.Done()

			w, err := watch.NewFromFile(operation.Log(), operation.GetKubeconfigPath(), nil)
			Expect(err).ToNot(HaveOccurred())
			go func() {
				Expect(w.Start(ctx)).ToNot(HaveOccurred())
			}()

			err = watch.WaitForCacheSyncWithTimeout(w, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			testrunConfig.Watch = w

			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			run := testrunner.RunList{
				&testrunner.Run{
					Testrun:  tr,
					Metadata: &metadata.Metadata{},
				},
			}
			err = testrunner.ExecuteTestruns(operation.Log(), &testrunConfig, run, "test-")
			defer utils.DeleteTestrun(operation.Client(), run[0].Testrun)
			Expect(err).ToNot(HaveOccurred())
			Expect(run.HasErrors()).To(BeFalse())

			Expect(len(run)).To(Equal(1))
			Expect(run[0].Testrun.Status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))
		})

		It("should run 2 testruns", func() {
			ctx := context.Background()
			defer ctx.Done()

			w, err := watch.NewFromFile(operation.Log(), operation.GetKubeconfigPath(), nil)
			Expect(err).ToNot(HaveOccurred())
			go func() {
				Expect(w.Start(ctx)).ToNot(HaveOccurred())
			}()

			err = watch.WaitForCacheSyncWithTimeout(w, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			testrunConfig.Watch = w

			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr2 := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			run := testrunner.RunList{
				&testrunner.Run{
					Testrun:  tr,
					Metadata: &metadata.Metadata{},
				},
				&testrunner.Run{
					Testrun:  tr2,
					Metadata: &metadata.Metadata{},
				},
			}
			err = testrunner.ExecuteTestruns(operation.Log(), &testrunConfig, run, "test-")
			defer utils.DeleteTestrun(operation.Client(), run[0].Testrun)
			defer utils.DeleteTestrun(operation.Client(), run[1].Testrun)
			Expect(err).ToNot(HaveOccurred())
			Expect(run.HasErrors()).To(BeFalse())

			Expect(len(run)).To(Equal(2))
			Expect(run[0].Testrun.Status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))
			Expect(run[1].Testrun.Status.Phase).To(Equal(tmv1beta1.PhaseStatusSuccess))
		})

	})

})
