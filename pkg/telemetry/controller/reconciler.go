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

package controller

import (
	"context"
	"fmt"
	telv1beta1 "github.com/gardener/test-infra/pkg/apis/telemetry/v1beta1"
	telmgr "github.com/gardener/test-infra/pkg/telemetry/manager"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type telemetryReconciler struct {
	client client.Client
	scheme *runtime.Scheme
	logger logr.Logger

	controllerManager telmgr.Manager
}

func (r *telemetryReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	defer ctx.Done()

	log := r.logger.WithValues("name", request.Name, "namespace", request.Namespace)

	st := &telv1beta1.ShootsMeasurement{}
	if err := r.client.Get(ctx, request.NamespacedName, st); err != nil {
		log.V(10).Info("unable to get ShootsMeasurement from cluster", "error", err.Error())
		return reconcile.Result{}, err
	}

	if st.DeletionTimestamp != nil {
		return r.delete(ctx, log, st)
	}

	if st.Status.Phase == telv1beta1.TelemetryPhaseCompleted {
		return reconcile.Result{Requeue: false}, nil
	}

	if addFinalizer(st) {
		if err := r.client.Update(ctx, st); err != nil {
			log.Error(err, "unable to add finalizer")
		}
		return reconcile.Result{}, nil
	}

	//if st.Generation == st.Status.ObservedGeneration {
	//	return reconcile.Result{Requeue: false}, nil
	//}

	log.Info("Reconcile")

	// add the shoots to the controller if no responsible controller is set
	controllerKey, err := r.controllerManager.MonitorShoots(ctx, client.ObjectKey{Name: st.Spec.GardenerSecretRef, Namespace: st.Namespace}, st.Spec.Shoots)
	if err != nil {
		log.Error(err, "unable to monitor shoots")
		st.Status.Phase = telv1beta1.TelemetryPhaseError
		st.Status.Message = fmt.Sprintf("Unable to add shoots to telemetry controller: %s", err.Error())
		return r.updateStatus(ctx, st, nil)
	}
	st.Status.Controller = controllerKey
	st.Status.Phase = telv1beta1.TelemetryPhaseRunning

	if metav1.HasAnnotation(st.ObjectMeta, telv1beta1.AnnotationStop) {
		if err := r.stop(st); err != nil {
			log.Error(err, "unable to stop measurement")
			return r.updateStatus(ctx, st, nil)
		}
	}

	return r.updateStatus(ctx, st, nil)
}

func (r *telemetryReconciler) stop(st *telv1beta1.ShootsMeasurement) error {
	if st.Status.Data == nil {
		st.Status.Data = make([]telv1beta1.ShootMeasurementData, 0)
	}
	for _, shoot := range st.Spec.Shoots {
		fig, err := r.controllerManager.StopAndAnalyze(st.Status.Controller, shoot)
		if err != nil {
			st.Status.Phase = telv1beta1.TelemetryPhaseError
			st.Status.Message = fmt.Sprintf("unable to get measurement for %s", shoot.String())
			return err
		}
		st.Status.Data = append(st.Status.Data, telv1beta1.ShootMeasurementData{
			Shoot:                 shoot,
			Provider:              fig.Provider,
			Seed:                  fig.Seed,
			CountUnhealthyPeriods: fig.CountUnhealthyPeriods,
			CountRequests:         fig.CountRequests,
			CountTimeouts:         fig.CountTimeouts,
			DownPeriods:           fig.DownPeriods,
			ResponseTimeDuration:  fig.ResponseTimeDuration,
		})
	}

	st.Status.Phase = telv1beta1.TelemetryPhaseCompleted
	st.Status.Message = fmt.Sprintf("Successfully measures %d shoots", len(st.Spec.Shoots))
	return nil
}

func (r *telemetryReconciler) delete(ctx context.Context, log logr.Logger, st *telv1beta1.ShootsMeasurement) (reconcile.Result, error) {

	if st.Status.Phase != telv1beta1.TelemetryPhaseCompleted {
		if err := r.stop(st); err != nil {
			log.Error(err, "unable to stop measurement")
			return r.updateStatus(ctx, st, nil)
		}
	}

	// remove finalizer and update
	stFinalizers := sets.NewString(st.Finalizers...)
	stFinalizers.Delete(telv1beta1.SchemeGroupVersion.Group)
	st.Finalizers = stFinalizers.UnsortedList()
	if err := r.client.Update(ctx, st); err != nil {
		log.Error(err, "unable to add finalizer")
	}
	return reconcile.Result{}, nil
}

func (r *telemetryReconciler) updateStatus(ctx context.Context, st *telv1beta1.ShootsMeasurement, err error) (reconcile.Result, error) {
	st.Status.ObservedGeneration = st.Generation
	if err := r.client.Status().Update(ctx, st); err != nil {
		r.logger.WithValues("name", st.Name, "namespace", st.Namespace).Error(err, "unable to update status")
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, err
}

// addFinalizer adds the controllers finalizer to the measurement and returns true if the finalizer was added
func addFinalizer(st *telv1beta1.ShootsMeasurement) bool {
	// add finalizers if not yet set
	stFinalizers := sets.NewString(st.Finalizers...)
	if stFinalizers.Has(telv1beta1.SchemeGroupVersion.Group) {
		return false
	}
	stFinalizers.Insert(telv1beta1.SchemeGroupVersion.Group)
	st.Finalizers = stFinalizers.UnsortedList()
	return true
}
