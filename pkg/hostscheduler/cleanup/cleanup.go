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

package cleanup

import (
	"context"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/go-logr/logr"
)

func CleanResources(ctx context.Context, logger logr.Logger, k8sClient kubernetes.Interface) error {
	if err := CleanWebhooks(ctx, logger, k8sClient.Client()); err != nil {
		return err
	}
	logger.Info("Cleaned Webhooks...")
	if err := CleanExtendedAPIs(ctx, logger, k8sClient.Client()); err != nil {
		return err
	}
	logger.Info("Cleaned Extended API...")
	if err := CleanKubernetesResources(ctx, logger, k8sClient.Client()); err != nil {
		return err
	}
	logger.Info("Cleaned Kubernetes resources...")

	return nil
}
