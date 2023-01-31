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
	"sync"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
)

var _ = Describe("Watch Cache Informer", func() {

	var (
		opts = &watch.Options{
			InformerType: watch.CachedInformerType,
		}
		ctx    context.Context
		cancel context.CancelFunc
		wg     sync.WaitGroup
		w      watch.Watch
	)

	BeforeEach(func(sCtx SpecContext) {
		ctx, cancel = context.WithCancel(context.Background())
		tr = &tmv1beta1.Testrun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
				Annotations: map[string]string{
					"test": "true",
				},
			},
		}
		err := fakeClient.Create(ctx, tr)
		Expect(err).ToNot(HaveOccurred())

		wg = sync.WaitGroup{}
		w, err = watch.New(logr.Discard(), restConfig, opts)
		Expect(err).ToNot(HaveOccurred())
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := w.Start(ctx)
			Expect(err).ToNot(HaveOccurred())
		}()
		err = watch.WaitForCacheSyncWithTimeout(w, 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())
	}, NodeTimeout(5*time.Second))

	AfterEach(func(sCtx SpecContext) {
		defer cancel()
		err := fakeClient.Delete(context.TODO(), tr)
		Expect(err).ToNot(HaveOccurred())

		cancel()
		wg.Wait()
	}, NodeTimeout(5*time.Second))

	It("watch should timeout after 2 seconds", func() {
		startTime := time.Now()
		err := w.WatchUntil(2*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) { return false, nil })
		Expect(err).To(HaveOccurred())
		Expect(time.Since(startTime).Seconds()).To(BeNumerically("~", 2, 0.01))
	})

	It("watch should reconcile once", func(sCtx SpecContext) {
		sendWG := sync.WaitGroup{}
		sendWG.Add(1)
		go func() {
			defer sendWG.Done()
			time.Sleep(1 * time.Second)
			By("updating testrun annotation")
			tr.Annotations["test"] = "false"
			err := fakeClient.Update(ctx, tr)
			Expect(err).ToNot(HaveOccurred())
		}()

		count := 0
		err := w.WatchUntil(10*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			if val, ok := tr.Annotations["test"]; ok && val == "false" {
				count++
				return true, nil
			}
			return false, nil
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(1))
		sendWG.Wait()
	}, NodeTimeout(12*time.Second))

	It("watch should get triggered for 10 changes", func(sCtx SpecContext) {
		sendWG := sync.WaitGroup{}
		sendWG.Add(1)
		go func() {
			defer sendWG.Done()
			defer GinkgoRecover()
			for i := 0; i < 10; i++ {
				time.Sleep(500 * time.Millisecond)
				tr.Annotations["rec"] = fmt.Sprintf("%d", i)
				err := fakeClient.Update(ctx, tr)
				Expect(err).ToNot(HaveOccurred())
			}
		}()

		count := 0
		err := w.WatchUntil(10*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			num, ok := tr.Annotations["rec"]
			if !ok {
				return false, nil
			}
			if num == strconv.Itoa(count) {
				count++
			}

			if count == 10 {
				return true, nil
			}
			return false, nil
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(10))
		sendWG.Wait()
	}, NodeTimeout(12*time.Second))

	It("watch should return an error if the watch function returns an error", func() {
		sendWG := sync.WaitGroup{}
		sendWG.Add(1)
		go func() {
			defer sendWG.Done()
			time.Sleep(10 * time.Millisecond)
			tr.Annotations["rec"] = "true"
			err := fakeClient.Update(ctx, tr)
			Expect(err).ToNot(HaveOccurred())
		}()

		err := w.WatchUntil(5*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			Expect(tr.Annotations).To(HaveKeyWithValue("test", "true"))
			return false, errors.New("error")
		})
		Expect(err).To(HaveOccurred())
		sendWG.Wait()
	})
})
