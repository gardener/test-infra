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
	"path/filepath"

	"github.com/gardener/gardener-resource-manager/pkg/health"
	"github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/chartrenderer"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/apis/config"
)

// checkResourceManager checks if a resource manager ist deployed
func (e *DependencyEnsurer) checkResourceManager(ctx context.Context, namespace string) error {
	deployment := &appsv1.Deployment{}
	if err := e.client.Get(ctx, client.ObjectKey{Name: config.ResourceManagerDeploymentName, Namespace: namespace}, deployment); err != nil {
		return err
	}
	return health.CheckDeployment(deployment)
}

// createManagedResource creates or updates a managed resource
func (e *DependencyEnsurer) createManagedResource(ctx context.Context, namespace, name string, renderer chartrenderer.Interface, chartName string, chartValues map[string]interface{}, injectedLabels map[string]string) error {
	return controller.CreateManagedResourceFromFileChart(
		ctx, e.client, namespace, name, "",
		renderer, filepath.Join(config.ChartsPath, chartName), chartName,
		chartValues, injectedLabels,
	)
}
