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
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/chart"
	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testmachinery/imagevector"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// checkResourceManager checks if a resource manager ist deployed
func (e *DependencyEnsurer) ensureObjectStore(ctx context.Context, namespace string, s3 *config.S3) error {
	e.log.Info("Ensuring object store")
	// ensure secret deployment
	if err := e.ensureS3Secret(ctx, s3, namespace); err != nil {
		return err
	}

	if s3.Server.Minio != nil {
		if err := e.validateMinioDeployment(ctx, namespace, s3); err != nil {
			return err
		}
		return e.ensureMinio(ctx, namespace, s3)
	}

	return nil
}

// validateMinioDeployment validates if the minio deployment method has not changed
func (e *DependencyEnsurer) validateMinioDeployment(ctx context.Context, namespace string, s3 *config.S3) error {
	// check if the minio deployment has changed (distributed or not)
	sts := &appsv1.StatefulSet{}
	if err := e.client.Get(ctx, client.ObjectKey{Name: config.MinioDeploymentName, Namespace: namespace}, sts); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// minio is distributed which means that the replicas are greater than 1
	if s3.Server.Minio.Distributed && *sts.Spec.Replicas > 1 {
		return nil
	}

	// minio runs as single deployment
	if *sts.Spec.Replicas == 1 {
		return nil
	}

	return errors.New("Deployment method of minio cannot be changed")
}

func (e *DependencyEnsurer) ensureMinio(ctx context.Context, namespace string, s3 *config.S3) error {
	e.log.Info("Ensuring minio deployment")
	values := map[string]interface{}{
		"minio": map[string]interface{}{
			"name": config.MinioDeploymentName,
			"distributed": map[string]interface{}{
				"enabled": s3.Server.Minio.Distributed,
			},
			"ingress": map[string]interface{}{
				"enabled": s3.Server.Minio.Ingress.Enabled,
				"host":    s3.Server.Minio.Ingress.Host,
			},
			"service": map[string]interface{}{
				"name": config.MinioServiceName,
				"port": config.MinioServicePort,
			},
		},
		"bucketName": s3.BucketName,
		"secret": map[string]string{
			"name": config.S3SecretName,
		},
	}

	if s3.Server.Minio.ChartValues != nil {
		additionalValues := map[string]interface{}{}
		if err := json.Unmarshal(s3.Server.Minio.ChartValues, &additionalValues); err != nil {
			return err
		}
		values = utils.MergeMaps(additionalValues, values)
	}

	values, err := chart.InjectImages(values, imagevector.ImageVector(), []string{
		config.MinioImageName,
	})
	if err != nil {
		return fmt.Errorf("failed to find image version %v", err)
	}

	err = e.createManagedResource(ctx, namespace, config.MinioManagedResourceName, e.renderer,
		config.MinioChartName, values, nil)
	if err != nil {
		return errors.Wrap(err, "unable to create managed resource")
	}

	// set the endpoint to use the internal minio service
	if len(s3.Server.Endpoint) == 0 {
		s3.Server.Endpoint = fmt.Sprintf("%s.%s.svc.cluster.local:%d", config.MinioServiceName, namespace, config.MinioServicePort)
	}

	return nil
}

func (e *DependencyEnsurer) ensureS3Secret(ctx context.Context, s3 *config.S3, namespace string) error {
	secret := &corev1.Secret{}
	secret.Name = config.S3SecretName
	secret.Namespace = namespace

	_, err := controllerruntime.CreateOrUpdate(ctx, e.client, secret, func() error {
		secret.StringData = map[string]string{
			"accessKey": s3.AccessKey,
			"secretKey": s3.SecretKey,
		}
		return nil
	})
	return err
}
