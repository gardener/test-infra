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
	"errors"
	argov1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testmachinery Watch Suite")
}

var _ = Describe("Watch", func() {

	var (
		tr         *tmv1beta1.Testrun
		tmScheme   *runtime.Scheme
		fakeClient client.Client
	)

	BeforeEach(func() {
		tmScheme = runtime.NewScheme()
		err := tmv1beta1.AddToScheme(tmScheme)
		Expect(err).ToNot(HaveOccurred())
		err = argov1alpha1.AddToScheme(tmScheme)
		Expect(err).ToNot(HaveOccurred())

		tr = &tmv1beta1.Testrun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
				Annotations: map[string]string{
					"test": "true",
				},
			},
		}
		fakeClient = fake.NewFakeClientWithScheme(tmScheme, tr)
	})

	It("watch should timeout after 5 seconds", func() {
		w, err := watch.New(log.NullLogger{}, nil)
		Expect(err).ToNot(HaveOccurred())

		startTime := time.Now()
		err = w.WatchUntil(2*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) { return false, nil })
		Expect(err).To(HaveOccurred())
		Expect(time.Now().Sub(startTime).Seconds()).To(BeNumerically("~", 2, 0.01))
	})

	It("watch should reconcile once", func() {
		w, err := watch.New(log.NullLogger{}, fakeClient)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			time.Sleep(3 * time.Second)
			_, err := w.Reconcile(reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "test", Namespace: "test"},
			})
			Expect(err).ToNot(HaveOccurred())
		}()

		count := 0
		err = w.WatchUntil(5*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			Expect(tr.Annotations).To(HaveKeyWithValue("test", "true"))
			count++
			return true, nil
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(1))
	})

	It("watch should reconcile 10 times", func() {
		w, err := watch.New(log.NullLogger{}, fakeClient)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			for i := 0; i < 10; i++ {
				time.Sleep(10 * time.Millisecond)
				_, err := w.Reconcile(reconcile.Request{
					NamespacedName: types.NamespacedName{Name: "test", Namespace: "test"},
				})
				Expect(err).ToNot(HaveOccurred())
			}
		}()

		count := 0
		err = w.WatchUntil(5*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			Expect(tr.Annotations).To(HaveKeyWithValue("test", "true"))
			count++

			if count == 10 {
				return true, nil
			}
			return false, nil
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(10))
	})

	It("watch should return an error if the watch function returns an error", func() {
		w, err := watch.New(log.NullLogger{}, fakeClient)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			time.Sleep(10 * time.Millisecond)
			_, err := w.Reconcile(reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "test", Namespace: "test"},
			})
			Expect(err).ToNot(HaveOccurred())
		}()

		count := 0
		err = w.WatchUntil(5*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			Expect(tr.Annotations).To(HaveKeyWithValue("test", "true"))
			count++
			return false, errors.New("error")
		})
		Expect(err).To(HaveOccurred())
		Expect(count).To(Equal(1))
	})
})
