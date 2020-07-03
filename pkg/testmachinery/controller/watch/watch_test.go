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
	"fmt"
	"testing"
	"time"

	argov1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/gardener/test-infra/pkg/apis/testmachinery"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testmachinery Watch Suite")
}

var _ = Describe("Watch", func() {

	var (
		testenv    *envtest.Environment
		restConfig *rest.Config
		tr         *tmv1beta1.Testrun
		fakeClient client.Client
	)

	BeforeSuite(func() {
		var err error
		crd := &v1beta1.CustomResourceDefinition{}
		crd.Name = "testruns.testmachinery.sapcloud.io"
		crd.Spec.Group = testmachinery.GroupName
		crd.Spec.Scope = v1beta1.NamespaceScoped
		crd.Spec.Names = v1beta1.CustomResourceDefinitionNames{
			Kind:     "Testrun",
			Plural:   "testruns",
			Singular: "testrun",
		}
		crd.Spec.Versions = []v1beta1.CustomResourceDefinitionVersion{{
			Name:    "v1beta1",
			Served:  true,
			Storage: true,
		}}
		crd.Spec.Subresources = &v1beta1.CustomResourceSubresources{
			Status: &v1beta1.CustomResourceSubresourceStatus{},
		}
		testenv = &envtest.Environment{
			CRDs: []runtime.Object{crd},
		}

		restConfig, err = testenv.Start()
		Expect(err).ToNot(HaveOccurred())

		tmScheme := runtime.NewScheme()
		err = tmv1beta1.AddToScheme(tmScheme)
		Expect(err).ToNot(HaveOccurred())
		err = argov1alpha1.AddToScheme(tmScheme)
		Expect(err).ToNot(HaveOccurred())
		fakeClient, err = client.New(restConfig, client.Options{Scheme: tmScheme})
		Expect(err).ToNot(HaveOccurred())
	})

	AfterSuite(func() {
		Expect(testenv.Stop()).ToNot(HaveOccurred())
	})

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

		w, err := watch.New(log.NullLogger{}, restConfig, nil)
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
		w, err := watch.New(log.NullLogger{}, restConfig, nil)
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
		err = w.WatchUntil(5*time.Second, "test", "test", func(tr *tmv1beta1.Testrun) (bool, error) {
			Expect(tr.Annotations).To(HaveKeyWithValue("test", "false"))
			count++
			return true, nil
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(1))
	})

	It("watch should reconcile 10 times", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		w, err := watch.New(log.NullLogger{}, restConfig, nil)
		Expect(err).ToNot(HaveOccurred())
		go func() {
			err := w.Start(ctx.Done())
			Expect(err).ToNot(HaveOccurred())
		}()

		err = watch.WaitForCacheSyncWithTimeout(w, 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		go func() {
			for i := 0; i < 10; i++ {
				time.Sleep(10 * time.Millisecond)
				tr.Annotations["rec"] = fmt.Sprintf("%d", i)
				err := fakeClient.Update(ctx, tr)
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		w, err := watch.New(log.NullLogger{}, restConfig, nil)
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
