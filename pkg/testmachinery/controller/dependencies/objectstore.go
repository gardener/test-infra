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

	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"

	"github.com/gardener/test-infra/pkg/apis/config"
)

// checkResourceManager checks if a resource manager ist deployed
func (e *DependencyEnsurer) ensureObjectStore(ctx context.Context, namespace string, s3 *config.S3) error {
	e.log.Info("Ensuring object store")
	// ensure secret deployment
	if err := e.ensureS3Secret(ctx, s3, namespace); err != nil {
		return err
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
