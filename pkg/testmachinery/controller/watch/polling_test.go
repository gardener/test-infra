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

package watch_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
)

var _ = Describe("Watch Polling Informer", func() {

	var (
		pollInterval = time.Second
		options      = &watch.Options{
			InformerType: watch.PollingInformerType,
			PollInterval: &pollInterval,
		}
	)

	BeforeEach(func() {
		tr = &tmv1beta1.Testrun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
				Annotations: map[string]string{
					"test": "true",
				},
			},
		}
		err := fakeClient.Create(context.TODO(), tr)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		err := fakeClient.Delete(context.TODO(), tr)
		Expect(err).ToNot(HaveOccurred())
	})

	It("watch should timeout after 2 seconds", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		w, err := watch.New(log.NullLogger{}, restConfig, options)
		Expect(err).ToNot(HaveOccurred())
		go func() {
			err := w.Start(ctx.Done())
			Expect(err).ToNot(HaveOccurred())
		}()

		err = watch.WaitForCacheSyncWithTimeout(w, 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		startTime := time.Now()
		err = w.WatchUntil(2*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) { return false, nil })
		Expect(err).To(HaveOccurred())
		Expect(time.Now().Sub(startTime).Seconds()).To(BeNumerically("~", 2, 0.01))
	})

	It("watch should reconcile once", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		w, err := watch.New(log.NullLogger{}, restConfig, options)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			err := w.Start(ctx.Done())
			Expect(err).ToNot(HaveOccurred())
		}()

		err = watch.WaitForCacheSyncWithTimeout(w, 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			time.Sleep(1 * time.Second)
			By("updating testrun annotation")
			tr.Annotations["test"] = "false"
			err := fakeClient.Update(ctx, tr)
			Expect(err).ToNot(HaveOccurred())
		}()

		count := 0
		err = w.WatchUntil(10*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			if val, ok := tr.Annotations["test"]; ok && val == "false" {
				count++
				return true, nil
			}
			return false, nil
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(1))
	})

	It("watch should reconcile until the latest update was received", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		w, err := watch.New(log.NullLogger{}, restConfig, options)
		Expect(err).ToNot(HaveOccurred())
		go func() {
			err := w.Start(ctx.Done())
			Expect(err).ToNot(HaveOccurred())
		}()

		err = watch.WaitForCacheSyncWithTimeout(w, 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			defer GinkgoRecover()
			for i := 0; i <= 10; i++ {
				time.Sleep(1 * time.Second)
				tr.Annotations["rec"] = fmt.Sprintf("%d", i)
				err := fakeClient.Update(ctx, tr)
				Expect(err).ToNot(HaveOccurred())
			}
		}()

		err = w.WatchUntil(50*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			num, ok := tr.Annotations["rec"]
			if !ok {
				return false, nil
			}

			if num == strconv.Itoa(10) {
				return true, nil
			}
			return false, nil
		})
		Expect(err).ToNot(HaveOccurred())
	})

	It("watch should return an error if the watch function returns an error", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		w, err := watch.New(log.NullLogger{}, restConfig, options)
		Expect(err).ToNot(HaveOccurred())
		go func() {
			err := w.Start(ctx.Done())
			Expect(err).ToNot(HaveOccurred())
		}()

		err = watch.WaitForCacheSyncWithTimeout(w, 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			time.Sleep(10 * time.Millisecond)
			tr.Annotations["rec"] = "true"
			err := fakeClient.Update(ctx, tr)
			Expect(err).ToNot(HaveOccurred())
		}()

		err = w.WatchUntil(5*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			Expect(tr.Annotations).To(HaveKeyWithValue("test", "true"))
			return false, errors.New("error")
		})
		Expect(err).To(HaveOccurred())
	})
})
