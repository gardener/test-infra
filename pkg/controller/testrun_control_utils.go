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
	"context"
	"strings"

	"github.com/gardener/test-infra/pkg/testmachinery"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *TestrunReconciler) getImagePullSecrets(ctx context.Context) []string {
	configMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: testmachinery.ConfigMapName, Namespace: testmachinery.GetConfig().Namespace}, configMap)
	if err != nil {
		log.Warnf("Cannot fetch Testmachinery config: %s", err.Error())
		return nil
	}

	pullSecretNames := configMap.Data["secrets.PullSecrets"]

	if pullSecretNames == "" {
		return nil
	}

	return strings.Split(pullSecretNames, ",")
}
