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

package controller

import (
	"context"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconcile handles various testrun events like crete, update and delete.
func (r *TestrunReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()

	tr := &tmv1beta1.Testrun{}
	err := r.Get(ctx, request.NamespacedName, tr)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{Requeue: false}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if tr.DeletionTimestamp != nil {
		log.Debug("Deletion: cause testrun")
		return r.deleteTestrun(ctx, tr)
	}

	///////////////
	// RECONCILE //
	///////////////

	foundWf := &argov1.Workflow{}
	err = r.Get(ctx, types.NamespacedName{Name: getWorkflowName(tr), Namespace: tr.Namespace}, foundWf)
	if err != nil && !errors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	if err != nil && errors.IsNotFound(err) && tr.Status.CompletionTime == nil {
		wf, err := r.createWorkflow(ctx, tr)
		if err != nil {
			log.Errorf("Error creating workflow for testrun %s\n%s", tr.Name, err.Error())
			return reconcile.Result{}, err
		}
		log.Infof("Creating workflow %s in namespace %s", wf.Name, wf.Namespace)
		err = r.Create(ctx, wf)
		if err != nil {
			log.Error(err)
			return reconcile.Result{Requeue: true}, err
		}

		tr.Status.Workflow = wf.Name
		tr.Status.Phase = tmv1beta1.PhaseStatusRunning

		// add finalizers for testrun
		trFinalizers := sets.NewString(tr.Finalizers...)
		if !trFinalizers.Has(tmv1beta1.SchemeGroupVersion.Group) {
			trFinalizers.Insert(tmv1beta1.SchemeGroupVersion.Group)
		}
		if !trFinalizers.Has(metav1.FinalizerDeleteDependents) {
			trFinalizers.Insert(metav1.FinalizerDeleteDependents)
		}
		tr.Finalizers = trFinalizers.UnsortedList()

		err = r.Update(ctx, tr)
		if err != nil {
			log.Errorf("Error updating testrun %s: %s", tr.Name, err.Error())
			return reconcile.Result{}, err
		}
	} else if err != nil {
		log.Errorf("Cannnot get workflow %s in namespace %s\n%s", getWorkflowName(tr), tr.Namespace, err.Error())
		return reconcile.Result{}, err
	}

	if tr.Status.CompletionTime != nil {
		if foundWf.DeletionTimestamp != nil {
			log.Debug("Deletion: cause workflow")
			return r.deleteTestrun(ctx, tr)
		}
		return reconcile.Result{}, err
	}

	return r.updateStatus(ctx, tr, foundWf)

}

func (r *TestrunReconciler) updateStatus(ctx context.Context, tr *tmv1beta1.Testrun, wf *argov1.Workflow) (reconcile.Result, error) {
	if !tr.Status.StartTime.Equal(&wf.Status.StartedAt) {
		tr.Status.StartTime = &wf.Status.StartedAt
	}
	if tr.Status.Phase == "" {
		tr.Status.Phase = tmv1beta1.PhaseStatusPending
	}
	if wf.Status.Completed() {

		err := r.completeTestrun(ctx, tr, wf)
		if err != nil {
			return reconcile.Result{}, nil
		}

		log.Infof("Testrun %s completed", tr.Name)
	}

	err := r.Update(ctx, tr)
	if err != nil {
		log.Errorf("Error updating Testrun status %s: %s", tr.Name, err.Error())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
