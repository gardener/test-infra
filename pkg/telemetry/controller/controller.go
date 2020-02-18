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
	"reflect"

	telv1beta1 "github.com/gardener/test-infra/pkg/apis/telemetry/v1beta1"
	telmgr "github.com/gardener/test-infra/pkg/telemetry/manager"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// NewTelemetryController creates new controller with the telemetry reconciler that watches Telemetry resources.
// Currently these resources are mainly the ShootTelemetry resource to monitor response and downtimes.
func NewTelemetryController(mgr manager.Manager, log logr.Logger, cacheDir string, maxConcurrentSyncs *int) (controller.Controller, error) {
	reconciler := &telemetryReconciler{
		client:            mgr.GetClient(),
		scheme:            mgr.GetScheme(),
		logger:            log.WithName("controller"),
		controllerManager: telmgr.New(log.WithName("telemetry-controller-manager"), mgr.GetClient(), cacheDir),
	}

	if err := telv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	opts := controller.Options{
		Reconciler: reconciler,
	}
	if maxConcurrentSyncs != nil {
		opts.MaxConcurrentReconciles = *maxConcurrentSyncs
	}

	c, err := controller.New("telemetry-controller", mgr, opts)
	if err != nil {
		return nil, err
	}

	if err := RegisterShootTelemetryWatch(c); err != nil {
		return nil, err
	}
	return c, nil
}

// RegisterShootTelemetryWatch registers event watches for shoot telemetry resources.
func RegisterShootTelemetryWatch(c controller.Controller) error {
	return c.Watch(&source.Kind{Type: &telv1beta1.ShootsMeasurement{}}, &handler.EnqueueRequestForObject{}, &predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if !reflect.DeepEqual(e.ObjectOld, e.ObjectNew) {
				return true
			}
			return false
		},
	})
}
