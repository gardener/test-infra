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

package reconciler

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/garbagecollection"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

func (r *TestmachineryReconciler) deleteTestrun(ctx context.Context, rCtx *reconcileContext) (reconcile.Result, error) {
	log := r.Logger.WithValues("testrun", types.NamespacedName{Name: rCtx.tr.Name, Namespace: rCtx.tr.Namespace})

	if finalizers := sets.NewString(rCtx.tr.GetFinalizers()...); !finalizers.Has(tmv1beta1.SchemeGroupVersion.Group) {
		return reconcile.Result{}, nil
	}

	foundWf := &argov1.Workflow{}
	err := r.Get(ctx, types.NamespacedName{Name: testmachinery.GetWorkflowName(rCtx.tr), Namespace: rCtx.tr.Namespace}, foundWf)
	if err != nil && !errors.IsNotFound(err) {
		return reconcile.Result{Requeue: true, RequeueAfter: 30 * time.Second}, err
	}
	if err == nil {
		log.Info("starting cleanup")
		if res, err := garbagecollection.GCWorkflowArtifacts(log, r.s3Client, foundWf); err != nil {
			return res, err
		}

		log.Info("deleting", "workflow", foundWf.Name)
		if removeFinalizer(foundWf, tmv1beta1.SchemeGroupVersion.Group) {
			err = r.Update(ctx, foundWf)
			if err != nil {
				log.Error(err, "unable to remove finalizer from workflow", "workflow", foundWf.Name)
				return reconcile.Result{Requeue: true, RequeueAfter: 30 * time.Second}, err
			}
		}

		if rCtx.tr.DeletionTimestamp == nil {
			log.Info("deleting", "testrun", rCtx.tr.Name)
			removeFinalizer(rCtx.tr, tmv1beta1.SchemeGroupVersion.Group)
			err = r.Delete(ctx, rCtx.tr)
			if err != nil {
				log.Error(err, "unable to delete workflow", "workflow", foundWf.Name)
				return reconcile.Result{}, err
			}
			return reconcile.Result{Requeue: true, RequeueAfter: 30 * time.Second}, err
		}
	}

	//remove finalizers
	if removeFinalizer(rCtx.tr, tmv1beta1.SchemeGroupVersion.Group) {
		err := r.Update(ctx, rCtx.tr)
		if err != nil {
			log.Error(err, "unable to remove finalizer from testrun")
			return reconcile.Result{Requeue: true, RequeueAfter: 30 * time.Second}, err
		}
	}
	return reconcile.Result{}, nil
}

func removeFinalizer(obj metav1.Object, finalizer string) bool {
	finalizers := sets.NewString(obj.GetFinalizers()...)
	if !finalizers.Has(finalizer) {
		return false
	}
	finalizers.Delete(tmv1beta1.SchemeGroupVersion.Group)
	obj.SetFinalizers(finalizers.UnsortedList())
	return true
}
