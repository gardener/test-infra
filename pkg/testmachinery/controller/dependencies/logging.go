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
	"path/filepath"

	"github.com/gardener/gardener/pkg/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	intconfig "github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testmachinery/imagevector"
)

func (e *DependencyEnsurer) ensureLoggingStack(ctx context.Context, config *intconfig.Logging) error {
	e.log.Info("Ensuring logging stack")

	if config == nil {
		return nil
	}

	e.log.V(3).Info("Ensuring Logging Namespace", "namespace", config.Namespace)
	ns := &corev1.Namespace{}
	ns.Name = config.Namespace
	if _, err := controllerutil.CreateOrUpdate(ctx, e.client, ns, func() error { return nil }); err != nil {
		return err
	}

	lokiImage, err := imagevector.ImageVector().FindImage(intconfig.LokiImageName)
	if err != nil {
		return errors.Wrapf(err, "unable to find image version for %s", intconfig.LokiImageName)
	}
	promtailImage, err := imagevector.ImageVector().FindImage(intconfig.PromtailImageName)
	if err != nil {
		return errors.Wrapf(err, "unable to find image version for %s", intconfig.PromtailImageName)
	}

	values := map[string]interface{}{
		"loki": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": lokiImage.Repository,
				"tag":        lokiImage.Tag,
			},
			"persistence": map[string]interface{}{
				"storageClassName": config.StorageClass,
			},
		},
		"promtail": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": promtailImage.Repository,
				"tag":        promtailImage.Tag,
			},
		},
	}

	if config.ChartValues != nil {
		additionalValues := map[string]interface{}{}
		if err := json.Unmarshal(config.ChartValues, &additionalValues); err != nil {
			return err
		}

		values = utils.MergeMaps(additionalValues, values)
	}

	loggingChart := &helmChart{
		Name:   intconfig.LoggingChartName,
		Path:   filepath.Join(intconfig.ChartsPath, intconfig.LoggingChartName),
		Images: []string{},
		Values: values,
	}

	err = e.createManagedResource(ctx,
		config.Namespace,
		intconfig.LoggingManagedResourceName,
		loggingChart,
		imagevector.ImageVector(),
	)
	if err != nil {
		e.log.Error(err, "unable to create logging managed resource")
		return err
	}
	return nil
}
