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
	"reflect"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// New creates a new Testmachinery controller for handling testruns and argo workflows.
func New(mgr manager.Manager) (*TestmachineryController, error) {

	tmv1beta1.AddToScheme(mgr.GetScheme())
	argov1.AddToScheme(mgr.GetScheme())

	reconciler := &TestrunReconciler{mgr.GetClient(), mgr.GetScheme()}
	c, err := controller.New("testmachinery-controller", mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return nil, err
	}

	return &TestmachineryController{c}, nil
}

// RegisterWatches registers event watches for testruns and testrun argo workflows.
func (tc *TestmachineryController) RegisterWatches() error {
	err := tc.Controller.Watch(&source.Kind{Type: &tmv1beta1.Testrun{}}, &handler.EnqueueRequestForObject{}, &predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if !reflect.DeepEqual(e.ObjectOld, e.ObjectNew) {
				return true
			}
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	})
	if err != nil {
		return err
	}

	err = tc.Controller.Watch(&source.Kind{Type: &argov1.Workflow{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &tmv1beta1.Testrun{},
	},
		&predicate.Funcs{
			DeleteFunc: func(_ event.DeleteEvent) bool {
				return false
			},
		})
	if err != nil {
		return err
	}
	return nil
}
