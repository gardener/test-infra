// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ttl_test

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
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
		ns.GenerateName = "test-"
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
		tr.Spec.TTLSecondsAfterFinished = ptr.To(int32(10))
		Expect(fakeClient.Create(ctx, tr)).To(Succeed())
		tr.Status.Phase = tmv1beta1.RunPhaseSuccess
		t := metav1.Now()
		tr.Status.CompletionTime = &t
		Expect(fakeClient.Status().Update(ctx, tr)).To(Succeed())

		res, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(tr)})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter.Seconds()).To(BeNumerically("~", 10.0, 1.2))

		Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(tr), tr)).To(Succeed())
		Expect(tr.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("should delete a testrun when its ttl is exceeded", func() {
		tr := &tmv1beta1.Testrun{}
		tr.Name = "exceeded"
		tr.Namespace = namespace
		tr.Spec.TTLSecondsAfterFinished = ptr.To(int32(10))
		Expect(fakeClient.Create(ctx, tr)).To(Succeed())
		tr.Status.Phase = tmv1beta1.RunPhaseSuccess
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
		tr.Spec.TTLSecondsAfterFinished = ptr.To(int32(10))
		Expect(fakeClient.Create(ctx, tr)).To(Succeed())
		tr.Status.Phase = tmv1beta1.RunPhaseError
		Expect(fakeClient.Status().Update(ctx, tr)).To(Succeed())

		res, err := controller.Reconcile(ctx, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(tr)})
		Expect(err).ToNot(HaveOccurred())
		Expect(res.RequeueAfter.Seconds()).To(BeNumerically("~", 10.0, 1))

		Expect(fakeClient.Get(ctx, client.ObjectKeyFromObject(tr), tr)).To(Succeed())
		Expect(tr.DeletionTimestamp.IsZero()).To(BeTrue())
	})

})
