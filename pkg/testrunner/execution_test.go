// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package testrunner_test

import (
	"math"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/testrunner"
)

var _ = Describe("Executor tests", func() {

	Context("basic", func() {
		It("should run a set of functions in serial without backoff", func(sCtx SpecContext) {
			executions := [3]*execution{}
			executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{
				Serial: true,
			})
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 3; i++ {
				e := newExecution(i)
				executions[i] = e
				f := func() {
					e.start = time.Now()
					time.Sleep(1 * time.Second)
				}
				executor.AddItem(f)
			}

			executor.Run()

			for i := 1; i < 3; i++ {
				e := executions[i]
				before := executions[i-1]
				Expect(e.start.After(before.start)).To(BeTrue())

				b := e.start.Sub(before.start)
				Expect(b.Seconds()).To(BeNumerically(">=", 1))
			}
		}, NodeTimeout(20*time.Second))

		It("should run all functions in parallel", func(sCtx SpecContext) {
			executions := [3]*execution{}
			executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{})
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 3; i++ {
				e := newExecution(i)
				executions[i] = e
				f := func() {
					e.start = time.Now()
					time.Sleep(1 * time.Second)
				}
				executor.AddItem(f)
			}

			executor.Run()

			for i := 1; i < 3; i++ {
				e := executions[i]
				before := executions[i-1]

				b := e.start.Sub(before.start)
				Expect(b.Seconds()).To(BeNumerically("~", 0, 0.1))
			}
		}, NodeTimeout(20*time.Second))

		It("should run 3 functions in serial with a backoff", func(sCtx SpecContext) {
			executions := [3]*execution{}
			executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{
				Serial:        true,
				BackoffBucket: 1,
				BackoffPeriod: 2 * time.Second,
			})
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 3; i++ {
				e := newExecution(i)
				executions[i] = e
				f := func() {
					e.start = time.Now()
				}
				executor.AddItem(f)
			}

			executor.Run()
			for i := 1; i < 3; i++ {
				e := executions[i]
				before := executions[i-1]
				Expect(e.start.After(before.start)).To(BeTrue())

				b := e.start.Sub(before.start)
				Expect(b.Seconds()).To(BeNumerically(">=", 2))
			}
		}, NodeTimeout(20*time.Second))

		It("should run 1 function in serial", func(sCtx SpecContext) {
			executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{
				Serial: true,
			})
			Expect(err).ToNot(HaveOccurred())

			var called int
			f := func() {
				called++
				time.Sleep(1 * time.Second)
			}
			executor.AddItem(f)

			executor.Run()
			Expect(called).To(Equal(1))
		}, NodeTimeout(20*time.Second))

		It("should run 3 functions in serial", func(sCtx SpecContext) {
			executions := [3]*execution{}
			executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{
				Serial: true,
			})
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 3; i++ {
				e := newExecution(i)
				executions[i] = e
				f := func() {
					e.start = time.Now()
					time.Sleep(1 * time.Second)
				}
				executor.AddItem(f)
			}

			executor.Run()
			for i := 1; i < 3; i++ {
				e := executions[i]
				before := executions[i-1]
				Expect(e.start.After(before.start)).To(BeTrue())

				b := e.start.Sub(before.start)
				Expect(b.Seconds()).To(BeNumerically(">=", 1))
			}
		}, NodeTimeout(20*time.Second))

		It("should run 6 functions in parallel with a backoff in a bucket of 2", func(sCtx SpecContext) {
			executions := [6]*execution{}
			executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{
				Serial:        false,
				BackoffBucket: 2,
				BackoffPeriod: 2 * time.Second,
			})
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 6; i++ {
				e := newExecution(i)
				executions[i] = e
				f := func() {
					e.start = time.Now()
					time.Sleep(1 * time.Second)
				}
				executor.AddItem(f)
			}

			executor.Run()

			expectExecutionsToHaveStartedInParallel(executions[0], executions[1])
			expectExecutionsToHaveStartedInParallel(executions[2], executions[3])
			expectExecutionsToHaveStartedInParallel(executions[4], executions[5])

			expectExecutionsToHaveBeenStartedAfter(executions[2], executions[0], 2)
			expectExecutionsToHaveBeenStartedAfter(executions[4], executions[2], 2)
		}, NodeTimeout(20*time.Second))

		It("should run 6 functions in serial with a backoff in a bucket of 2", func(sCtx SpecContext) {
			executions := [6]*execution{}
			executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{
				Serial:        true,
				BackoffBucket: 2,
				BackoffPeriod: 2 * time.Second,
			})
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 6; i++ {
				e := newExecution(i)
				executions[i] = e
				f := func() {
					e.start = time.Now()
					time.Sleep(1 * time.Second)
				}
				executor.AddItem(f)
			}

			executor.Run()

			expectExecutionsToHaveStartedInParallel(executions[0], executions[1])
			expectExecutionsToHaveStartedInParallel(executions[2], executions[3])
			expectExecutionsToHaveStartedInParallel(executions[4], executions[5])

			expectExecutionsToHaveBeenStartedAfter(executions[2], executions[0], 3)
			expectExecutionsToHaveBeenStartedAfter(executions[4], executions[2], 3)
		}, NodeTimeout(20*time.Second))
	})

	It("should add another test during execution", func(sCtx SpecContext) {
		executions := [3]*execution{}
		executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{
			Serial: true,
		})
		Expect(err).ToNot(HaveOccurred())

		addExecution := newExecution(4)

		for i := 0; i < 3; i++ {
			e := newExecution(i)
			executions[i] = e
			f := func() {
				e.start = time.Now()
				time.Sleep(5 * time.Second)
				if e.value == 2 {
					executor.AddItem(func() {
						addExecution.start = time.Now()
					})
				}
			}
			executor.AddItem(f)
		}

		executor.Run()

		Expect(addExecution.start.IsZero()).To(BeFalse())
		expectExecutionsToHaveBeenStartedAfter(addExecution, executions[2], 5)

	}, NodeTimeout(20*time.Second))

	It("should add another test during execution in parallel steps", func(sCtx SpecContext) {
		executions := [3]*execution{}
		executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{})
		Expect(err).ToNot(HaveOccurred())

		addExecution := newExecution(4)

		for i := 0; i < 3; i++ {
			e := newExecution(i)
			executions[i] = e
			f := func() {
				e.start = time.Now()
				time.Sleep(5 * time.Second)
				if e.value == 2 {
					executor.AddItem(func() {
						addExecution.start = time.Now()
					})
				}
			}
			executor.AddItem(f)
		}

		executor.Run()

		Expect(addExecution.start.IsZero()).To(BeFalse())
		expectExecutionsToHaveBeenStartedAfter(addExecution, executions[2], 5)
	}, NodeTimeout(20*time.Second))

	It("should add another test during execution in parallel steps that start immediately", func(sCtx SpecContext) {
		executions := [3]*execution{}
		executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{})
		Expect(err).ToNot(HaveOccurred())

		addExecution := newExecution(4)

		for i := 0; i < 3; i++ {
			e := newExecution(i)
			executions[i] = e
			f := func() {
				e.start = time.Now()

				if e.value == 1 {
					time.Sleep(5 * time.Second)
					executor.AddItem(func() {
						addExecution.start = time.Now()
					})
				} else {
					time.Sleep(2 * time.Second)
				}
			}
			executor.AddItem(f)
		}

		executor.Run()

		Expect(addExecution.start.IsZero()).To(BeFalse())
		expectExecutionsToHaveBeenStartedAfter(addExecution, executions[0], 5)
	}, NodeTimeout(20*time.Second))

	It("should add same test during execution in parallel steps", func(sCtx SpecContext) {
		executions := [3]*execution{}
		executor, err := testrunner.NewExecutor(logr.Discard(), testrunner.ExecutorConfig{})
		Expect(err).ToNot(HaveOccurred())

		for i := 0; i < 3; i++ {
			e := newExecution(i)
			executions[i] = e
			var f func()
			f = func() {
				e.start = time.Now()
				time.Sleep(5 * time.Second)
				if e.value == 1 {
					e.value = 3
					executor.AddItem(f)
				}
			}
			executor.AddItem(f)
		}

		executor.Run()

		Expect(executions[1].value).To(Equal(3))
		expectExecutionsToHaveBeenStartedAfter(executions[1], executions[2], 5)
	}, NodeTimeout(20*time.Second))

})

func expectExecutionsToHaveBeenStartedAfter(e1, e2 *execution, expDurationSeconds int) {
	d := e1.start.Sub(e2.start).Seconds()
	d = math.Round(d*10) / 10
	ExpectWithOffset(1, d).To(BeNumerically(">=", expDurationSeconds), "duration is %fs but expected %ds", d, expDurationSeconds)
}

func expectExecutionsToHaveStartedInParallel(e1, e2 *execution) {
	d := math.Abs(e1.start.Sub(e2.start).Seconds())
	ExpectWithOffset(1, d).To(BeNumerically("~", 0, 1), "duration is %fs but expected have been started in parallel", d)
}

func newExecution(i int) *execution {
	return &execution{value: i}
}

type execution struct {
	start time.Time
	value int
}
