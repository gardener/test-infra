// Copyright 2019 Copyright (c) 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package setup

import (
	"context"
	"os"

	authenticationv1alpha1 "github.com/gardener/gardener/pkg/apis/authentication/v1alpha1"
	"github.com/gardener/gardener/pkg/client/core/clientset/versioned/typed/core/v1beta1"
	"github.com/gardener/test-infra/integration-tests/e2e/config"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/utils/pointer"
)

// tokenExpiresAfterSeconds sets the token to expire after 24h. All tests running longer will be considered as failed.
const tokenExpiresAfterSeconds int64 = 86400

// setupKubeconfig requests an admin kubeconfig for a given shoot
func setupKubeconfig() error {

	// check, if a shoots/adminkubeconfig can be requested
	// see https://gardener.cloud/docs/gardener/usage/shoot_access/#shootsadminkubeconfig-subresource for details
	if config.GardenKubeconfigPath != "" && config.ProjectNamespace != "" && config.ShootName != "" {

		adminKubeconfigRequest := &authenticationv1alpha1.AdminKubeconfigRequest{
			Spec: authenticationv1alpha1.AdminKubeconfigRequestSpec{
				ExpirationSeconds: pointer.Int64(tokenExpiresAfterSeconds),
			},
		}

		gardenKubeconfigData, err := os.ReadFile(config.GardenKubeconfigPath)
		if err != nil {
			return err
		}
		gardenClientConfig, err := clientcmd.NewClientConfigFromBytes(gardenKubeconfigData)
		if err != nil {
			return err
		}
		restConfig, err := gardenClientConfig.ClientConfig()
		if err != nil {
			return err
		}
		gardenerRestClient, err := v1beta1.NewForConfig(restConfig)
		if err != nil {
			return err
		}
		adminKubeconfigRequest, err = gardenerRestClient.Shoots(config.ProjectNamespace).CreateAdminKubeconfigRequest(context.Background(), config.ShootName, adminKubeconfigRequest, metav1.CreateOptions{})
		if err != nil {
			return err
		}

		err = os.WriteFile(config.ShootKubeconfigPath, adminKubeconfigRequest.Status.Kubeconfig, 0755)
		if err != nil {
			return err
		}

		log.Info("shoot admin kubeconfig successfully obtained")
	}

	return nil
}
