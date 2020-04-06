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
	"encoding/json"
	"fmt"
	"github.com/gardener/gardener-resource-manager/pkg/apis/resources/v1alpha1"
	"github.com/gardener/gardener-resource-manager/pkg/health"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/chart"
	intconfig "github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/apis/config/validation"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/dependencies/configwatcher"
	tmhealth "github.com/gardener/test-infra/pkg/testmachinery/controller/health"
	"github.com/gardener/test-infra/pkg/testmachinery/imagevector"
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

	tmhealth.AddHealthCondition("bootsrap", b)

	return b, nil
}

// Start is only needed during startup to ensure all needed deployments are healthy
func (b *DependencyEnsurer) Start(ctx context.Context, mgr manager.Manager) error {
	var err error
	s := runtime.NewScheme()
	if err := scheme.AddToScheme(s); err != nil {
		return err
	}
	if err := v1alpha1.AddToScheme(s); err != nil {
		return err
	}

	b.client, err = client.New(mgr.GetConfig(), client.Options{Scheme: s})
	if err != nil {
		return err
	}

	if err := b.Reconcile(ctx, b.cw.GetConfiguration()); err != nil {
		return err
	}

	b.cw.InjectNotifyFunc(b.Reconcile)

	// start configwatch
	go func() {
		if err := b.cw.Start(ctx.Done()); err != nil {
			b.log.Error(err, "error while watching config")
		}
	}()

	return nil
}

// CheckHealth checks the current health of all deployed components
func (b *DependencyEnsurer) CheckHealth(ctx context.Context) error {
	config := b.cw.GetConfiguration()
	if config == nil {
		return nil
	}

	namespace := config.TestMachineryConfiguration.Namespace

	if err := b.checkResourceManager(ctx, namespace); err != nil {
		return err
	}

	if config.S3Configuration.Server.Minio != nil {
		mr := &v1alpha1.ManagedResource{}
		if err := b.client.Get(ctx, client.ObjectKey{Name: intconfig.ArgoManagedResourceName, Namespace: namespace}, mr); err != nil {
			return err
		}
		if err := health.CheckManagedResourceHealthy(mr); err != nil {
			return err
		}
	}

	mr := &v1alpha1.ManagedResource{}
	if err := b.client.Get(ctx, client.ObjectKey{Name: intconfig.ArgoManagedResourceName, Namespace: namespace}, mr); err != nil {
		return err
	}
	return health.CheckManagedResourceHealthy(mr)
}

// Reconcile ensures the correct state defined by the configuration.
func (b *DependencyEnsurer) Reconcile(ctx context.Context, config *intconfig.Configuration) error {
	b.log.Info("Ensuring bootstrap components")
	errs := validation.ValidateConfiguration(config)
	if len(errs) > 0 {
		return errs.ToAggregate()
	}

	namespace := config.TestMachineryConfiguration.Namespace

	if err := b.checkResourceManager(ctx, namespace); err != nil {
		b.log.Error(err, "resource manager not ready")
		return err
	}

	if err := b.ensureObjectStore(ctx, namespace, config.S3Configuration); err != nil {
		return err
	}

	if err := b.ensureArgo(ctx, namespace, config); err != nil {
		return err
	}

	return testmachinery.Setup(config)
}

func (b *DependencyEnsurer) ensureArgo(ctx context.Context, namespace string, config *intconfig.Configuration) error {
	b.log.Info("Ensuring argo deployment")
	values := map[string]interface{}{
		"argo": map[string]interface{}{
			"name": intconfig.ArgoWorkflowControllerDeploymentName,
		},
		"argoui": map[string]interface{}{
			"ingress": map[string]interface{}{
				"enabled": config.Argo.ArgoUI.Ingress.Enabled,
				"name":    intconfig.ArgoUIIngressName,
				"host":    config.Argo.ArgoUI.Ingress.Host,
			},
		},
		"objectStorage": map[string]interface{}{
			"bucketName": config.S3Configuration.BucketName,
			"endpoint":   config.S3Configuration.Server.Endpoint,
			"secret": map[string]string{
				"name": intconfig.S3SecretName,
			},
		},
	}

	if config.Argo.ChartValues != nil {
		additionalValues := map[string]interface{}{}
		if err := json.Unmarshal(config.Argo.ChartValues, &additionalValues); err != nil {
			return err
		}

		values = utils.MergeMaps(additionalValues, values)
	}

	values, err := chart.InjectImages(values, imagevector.ImageVector(), []string{
		intconfig.ArgoUIImageName,
		intconfig.ArgoWorkflowControllerImageName,
		intconfig.ArgoExecutorImageName,
	})
	if err != nil {
		return fmt.Errorf("failed to find image version %v", err)
	}

	err = b.createManagedResource(ctx, namespace, intconfig.ArgoManagedResourceName, b.renderer,
		intconfig.ArgoChartName, values, nil)
	if err != nil {
		b.log.Error(err, "unable to create managed resource")
		return err
	}
	return nil
}
