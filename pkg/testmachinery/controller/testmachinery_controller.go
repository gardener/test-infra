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
	"fmt"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/test-infra/pkg/apis/config"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/collector"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/reconciler"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/ttl"
	"github.com/gardener/test-infra/pkg/testmachinery/ghcache"
	"github.com/gardener/test-infra/pkg/util/s3"
)

// RegisterTestMachineryController creates new controller with the testmachinery reconciler that watches testruns and workflows
func RegisterTestMachineryController(mgr manager.Manager, log logr.Logger, config *config.Configuration) error {
	var (
		err      error
		collect  collector.Interface
		s3Client s3.Client
	)

	if err := tmv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}
	if err := argov1.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	if !config.TestMachinery.DisableCollector {
		collect, err = collector.New(ctrl.Log, mgr.GetClient(), testmachinery.GetElasticsearchConfiguration(), testmachinery.GetS3Configuration())
		if err != nil {
			return fmt.Errorf("unable to setup collector: %w", err)
		}
	}

	if !config.TestMachinery.Local {
		s3Client, err = s3.New(s3.FromConfig(testmachinery.GetS3Configuration()))
		if err != nil {
			return fmt.Errorf("unable to setup s3 client: %w", err)
		}
	}

	tmReconciler := reconciler.New(log.WithName("controller"), mgr.GetClient(), mgr.GetScheme(), s3Client, collect)
	bldr := ctrl.NewControllerManagedBy(mgr).
		For(&tmv1beta1.Testrun{}).
		Owns(&argov1.Workflow{})

	if config.Controller.MaxConcurrentSyncs != 0 {
		bldr.WithOptions(controller.Options{
			MaxConcurrentReconciles: config.Controller.MaxConcurrentSyncs,
		})
	}
	if err := bldr.Complete(tmReconciler); err != nil {
		return err
	}

	ghcache.InitGitHubCache(config.GitHub.Cache)

	if !config.Controller.TTLController.Disable {
		return ttl.AddControllerToManager(log, mgr, config.Controller.TTLController)
	}
	return nil
}
