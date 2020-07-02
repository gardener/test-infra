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

package dependencies

import (
	"context"
	"github.com/gardener/gardener-resource-manager/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener-resource-manager/pkg/health"
	"github.com/gardener/gardener/pkg/chartrenderer"
	intconfig "github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/apis/config/validation"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/dependencies/configwatcher"
	tmhealth "github.com/gardener/test-infra/pkg/testmachinery/controller/health"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/helm/pkg/engine"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// DependencyEnsurer reconciles all dependencies that are needed by the testmachinery
type DependencyEnsurer struct {
	client client.Client
	log    logr.Logger

	cw *configwatcher.ConfigWatcher

	renderer chartrenderer.Interface
}

var _ tmhealth.Condition = &DependencyEnsurer{}

// New returns a new dependency ensurer
func New(log logr.Logger, cw *configwatcher.ConfigWatcher) (*DependencyEnsurer, error) {
	b := &DependencyEnsurer{
		log:      log,
		cw:       cw,
		renderer: chartrenderer.New(engine.New(), nil),
	}

	tmhealth.AddHealthCondition("bootstrap", b)

	return b, nil
}

// Start is only needed during startup to ensure all needed deployments are healthy
func (e *DependencyEnsurer) Start(ctx context.Context, mgr manager.Manager) error {
	var err error
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		return err
	}
	if err := v1alpha1.AddToScheme(s); err != nil {
		return err
	}

	e.client, err = client.New(mgr.GetConfig(), client.Options{Scheme: s})
	if err != nil {
		return err
	}

	if err := e.Reconcile(ctx, e.cw.GetConfiguration()); err != nil {
		return err
	}

	e.cw.InjectNotifyFunc(e.Reconcile)

	// start configwatch
	go func() {
		if err := e.cw.Start(ctx.Done()); err != nil {
			e.log.Error(err, "error while watching config")
		}
	}()

	return nil
}

// CheckHealth checks the current health of all deployed components
func (e *DependencyEnsurer) CheckHealth(ctx context.Context) error {
	config := e.cw.GetConfiguration()
	if config == nil {
		return nil
	}

	namespace := config.TestMachinery.Namespace

	if err := e.checkResourceManager(ctx, namespace); err != nil {
		return err
	}

	if config.S3.Server.Minio != nil {
		mr := &v1alpha1.ManagedResource{}
		if err := e.client.Get(ctx, client.ObjectKey{Name: intconfig.ArgoManagedResourceName, Namespace: namespace}, mr); err != nil {
			return err
		}
		if err := health.CheckManagedResourceHealthy(mr); err != nil {
			return err
		}
	}

	if config.Observability.Logging != nil {
		mr := &v1alpha1.ManagedResource{}
		if err := e.client.Get(ctx, client.ObjectKey{Name: intconfig.LoggingManagedResourceName, Namespace: config.Observability.Logging.Namespace}, mr); err != nil {
			return err
		}
		if err := health.CheckManagedResourceHealthy(mr); err != nil {
			return err
		}
	}

	return nil
}

// Reconcile ensures the correct state defined by the configuration.
func (e *DependencyEnsurer) Reconcile(ctx context.Context, config *intconfig.Configuration) error {
	e.log.Info("Ensuring bootstrap components")
	errs := validation.ValidateConfiguration(config)
	if len(errs) > 0 {
		return errs.ToAggregate()
	}

	namespace := config.TestMachinery.Namespace

	if err := e.checkResourceManager(ctx, namespace); err != nil {
		e.log.Error(err, "resource manager not ready")
		return err
	}

	if err := e.ensureObjectStore(ctx, namespace, config.S3); err != nil {
		return err
	}

	if err := e.ensureArgo(ctx, namespace, config); err != nil {
		return err
	}

	if err := e.ensureReserveExcessCapacityPods(ctx, namespace, config.ReservedExcessCapacity); err != nil {
		return err
	}

	if err := e.ensureLoggingStack(ctx, config.Observability.Logging); err != nil {
		return err
	}

	return testmachinery.Setup(config)
}
