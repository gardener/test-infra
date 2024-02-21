// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testrunner_run_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
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
			Expect(run[0].Testrun.Status.Phase).To(Equal(tmv1beta1.StepPhaseSuccess))
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
			Expect(run[0].Testrun.Status.Phase).To(Equal(tmv1beta1.StepPhaseSuccess))
			Expect(run[1].Testrun.Status.Phase).To(Equal(tmv1beta1.StepPhaseSuccess))
		})

	})

})
