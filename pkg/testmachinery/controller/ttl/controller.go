// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ttl

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// Controller defines the ttl controller.
type Controller struct {
	log    logr.Logger
	scheme *runtime.Scheme
	client client.Client
}

// New creates a new ttl controller
func New(log logr.Logger, kubeClient client.Client, scheme *runtime.Scheme) *Controller {
	return &Controller{
		log:    log,
		scheme: scheme,
		client: kubeClient,
	}
}

func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	tr := &tmv1beta1.Testrun{}
	if err := c.client.Get(ctx, req.NamespacedName, tr); err != nil {
		return reconcile.Result{}, err
	}
	logger := c.log.WithValues("testrun", req.String())

	if !tr.DeletionTimestamp.IsZero() {
		// skip reconcile if testrun is already deleted.
		return reconcile.Result{}, nil
	}

	if tr.Spec.TTLSecondsAfterFinished == nil {
		logger.V(10).Info("testrun has not ttl set")
		return reconcile.Result{}, nil
	}

	if !tr.Status.Phase.Completed() {
		logger.V(7).Info("testrun still progressing")
		return reconcile.Result{}, nil
	}

	trTime := tr.Status.CompletionTime
	// the completion time should be set when the testrun is completed.
	// but if not we default to the creation time
	if trTime == nil {
		trTime = &tr.CreationTimestamp
	}

	// time when the ttl is expired
	ttlDuration := time.Duration(*tr.Spec.TTLSecondsAfterFinished) * time.Second
	ttlExpiredTime := trTime.Add(ttlDuration)

	if t := time.Since(ttlExpiredTime); t < 0 {
		logger.V(5).Info("testrun ttl not yet expired requeuing")
		return reconcile.Result{
			RequeueAfter: t * -1,
		}, nil
	}

	if err := c.client.Delete(ctx, tr); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
