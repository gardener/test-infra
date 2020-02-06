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
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/reconciler"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	"github.com/go-logr/logr"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// NewTestMachineryController creates new controller with the testmachinery reconciler that watches testruns and workflows
func NewTestMachineryController(mgr manager.Manager, log logr.Logger, maxConcurrentSyncs *int) (controller.Controller, error) {
	tmReconciler := reconciler.NewReconciler(mgr, log.WithName("controller"))
	c, err := New(mgr, tmReconciler, maxConcurrentSyncs)
	if err != nil {
		return nil, err
	}
	if err := RegisterDefaultWatches(c); err != nil {
		return nil, err
	}
	return c, nil
}

// NewWatchController creates new controller with th watch reconciler that watches testruns
func NewWatchController(mgr manager.Manager, log logr.Logger) (controller.Controller, watch.Watch, error) {
	w, err := watch.New(log.WithName("watch"), mgr.GetClient())
	if err != nil {
		return nil, nil, err
	}
	c, err := New(mgr, w, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := RegisterTestrunWatch(c); err != nil {
		return nil, nil, err
	}
	return c, w, nil
}

// New creates a new Testmachinery controller for handling testruns and argo workflows.
func New(mgr manager.Manager, r reconcile.Reconciler, maxConcurrentReconciles *int) (controller.Controller, error) {
	if err := tmv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}
	if err := argov1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	opts := controller.Options{
		Reconciler: r,
	}
	if maxConcurrentReconciles != nil {
		opts.MaxConcurrentReconciles = *maxConcurrentReconciles
	}

	c, err := controller.New("testmachinery-controller", mgr, opts)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// RegisterDefaultWatches registers event watches for testruns and testrun argo workflows.
func RegisterDefaultWatches(c controller.Controller) error {
	if err := RegisterTestrunWatch(c); err != nil {
		return err
	}
	return RegisterArgoWorkflowWatch(c)
}

// RegisterArgoWorkflowWatch registers event watches for argo workflows.
func RegisterArgoWorkflowWatch(c controller.Controller) error {
	return c.Watch(&source.Kind{Type: &argov1.Workflow{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &tmv1beta1.Testrun{},
	},
		&predicate.Funcs{
			DeleteFunc: func(_ event.DeleteEvent) bool {
				return false
			},
		})
}

// RegisterTestrunWatch registers event watches for testruns.
func RegisterTestrunWatch(c controller.Controller) error {
	return c.Watch(&source.Kind{Type: &tmv1beta1.Testrun{}}, &handler.EnqueueRequestForObject{}, &predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if !reflect.DeepEqual(e.ObjectOld, e.ObjectNew) {
				return true
			}
			return false
		},
	})
}
