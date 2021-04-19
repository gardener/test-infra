// Copyright 2021 Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package ttl_test

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/ttl"
)

var _ = Describe("Reconcile", func() {

	var (
		ctx        context.Context
		namespace  string
		controller reconcile.Reconciler
	)
	BeforeEach(func() {
		ctx = context.Background()
		ns := &corev1.Namespace{}
		ns.Name = fmt.Sprintf("test-%d", rand.Intn(1000))
		if err := fakeClient.Create(ctx, ns); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				Expect(err).ToNot(HaveOccurred())
			}
		}
		namespace = ns.Name

		controller = ttl.New(logr.Discard(), fakeClient, testmachinery.TestMachineryScheme)
	})

	AfterEach(func() {
		defer ctx.Done()
		ns := &corev1.Namespace{}
		ns.Name = namespace
		Expect(fakeClient.Delete(ctx, ns)).To(Succeed())
		Eventually(func() error {
			if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(ns), &corev1.Namespace{}); err != nil {
				if apierrors.IsNotFound(err) {
					return nil
				}
				return err
			}
			return nil
		}, 1*time.Minute, 10*time.Second).Should(Succeed())
	})

	It("should not delete a testrun when no ttl is set", func() {
		tr := &tmv1beta1.Testrun{}
		tr.Name = "no-ttl"
		tr.Namespace = namespace
		Expect(fakeClient.Create(ctx, tr)).To(Succeed())

		res, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(tr)})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter.Seconds()).To(Equal(0.0))

		Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(tr), tr)).To(Succeed())
		Expect(tr.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("should not delete a testrun and requeue when its ttl is no yet exceeded", func() {
		tr := &tmv1beta1.Testrun{}
		tr.Name = "not-exceeded"
		tr.Namespace = namespace
		tr.Spec.TTLSecondsAfterFinished = pointer.Int32Ptr(10)
		Expect(fakeClient.Create(ctx, tr)).To(Succeed())
		tr.Status.Phase = tmv1beta1.PhaseStatusSuccess
		t := metav1.Now()
		tr.Status.CompletionTime = &t
		Expect(fakeClient.Status().Update(ctx, tr)).To(Succeed())

		res, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(tr)})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter.Seconds()).To(BeNumerically("~", 10.0, 1))

		Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(tr), tr)).To(Succeed())
		Expect(tr.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("should delete a testrun when its ttl is exceeded", func() {
		tr := &tmv1beta1.Testrun{}
		tr.Name = "exceeded"
		tr.Namespace = namespace
		tr.Spec.TTLSecondsAfterFinished = pointer.Int32Ptr(10)
		Expect(fakeClient.Create(ctx, tr)).To(Succeed())
		tr.Status.Phase = tmv1beta1.PhaseStatusSuccess
		tr.Status.CompletionTime = &metav1.Time{Time: metav1.Now().Add(-20 * time.Second)}
		Expect(fakeClient.Status().Update(ctx, tr)).To(Succeed())

		res, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(tr)})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter.Seconds()).To(Equal(0.0))

		err = fakeClient.Get(ctx, client.ObjectKeyFromObject(tr), tr)
		if !apierrors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
		if err == nil {
			Expect(tr.DeletionTimestamp.IsZero()).To(BeFalse())
		}
	})

	It("should use the creation timestamp if no completion timestamp exists", func() {
		tr := &tmv1beta1.Testrun{}
		tr.Name = "no-ts"
		tr.Namespace = namespace
		tr.Spec.TTLSecondsAfterFinished = pointer.Int32Ptr(10)
		Expect(fakeClient.Create(ctx, tr)).To(Succeed())
		tr.Status.Phase = tmv1beta1.PhaseStatusError
		Expect(fakeClient.Status().Update(ctx, tr)).To(Succeed())

		res, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(tr)})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter.Seconds()).To(BeNumerically("~", 10.0, 1))

		Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(tr), tr)).To(Succeed())
		Expect(tr.DeletionTimestamp.IsZero()).To(BeTrue())
	})

})
