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

package testrun

import (
	"encoding/base64"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/util/strconf"
)

// ParseKubeconfigs parses the kubeconfigs defined in the testrun and returns respective configs and k8s secrets.
func ParseKubeconfigs(tr *tmv1beta1.Testrun) ([]*config.Element, []client.Object, error) {
	configs := make([]*config.Element, 0)
	secrets := make([]client.Object, 0)
	if tr.Spec.Kubeconfigs.Host != nil {
		if err := addKubeconfig(&configs, &secrets, tr, "host", tr.Spec.Kubeconfigs.Host); err != nil {
			return nil, nil, err
		}
	}
	if tr.Spec.Kubeconfigs.Gardener != nil {
		if err := addKubeconfig(&configs, &secrets, tr, "gardener", tr.Spec.Kubeconfigs.Gardener); err != nil {
			return nil, nil, err
		}
	}
	if tr.Spec.Kubeconfigs.Seed != nil {
		if err := addKubeconfig(&configs, &secrets, tr, "seed", tr.Spec.Kubeconfigs.Seed); err != nil {
			return nil, nil, err
		}
	}
	if tr.Spec.Kubeconfigs.Shoot != nil {
		if err := addKubeconfig(&configs, &secrets, tr, "shoot", tr.Spec.Kubeconfigs.Shoot); err != nil {
			return nil, nil, err
		}
	}

	return configs, secrets, nil
}

func addKubeconfig(configs *[]*config.Element, secrets *[]client.Object, tr *tmv1beta1.Testrun, name string, kubeconfig *strconf.StringOrConfig) error {
	kubeconfigPath := fmt.Sprintf("%s/%s.config", testmachinery.TM_KUBECONFIG_PATH, name)
	if kubeconfig.Type == strconf.String {

		rawKubeconfig, err := base64.StdEncoding.DecodeString(kubeconfig.String())
		if err != nil {
			return fmt.Errorf("unable to decode %s kubeconfig: %s", name, err.Error())
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", tr.Name, name),
				Namespace: tr.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion:         tr.GroupVersionKind().GroupVersion().String(),
						Kind:               tr.Kind,
						Name:               tr.GetName(),
						UID:                tr.GetUID(),
						BlockOwnerDeletion: pointer.BoolPtr(true),
						Controller:         pointer.BoolPtr(false),
					},
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"kubeconfig": rawKubeconfig,
			},
		}
		*secrets = append(*secrets, secret)

		*configs = append(*configs, config.NewElement(&tmv1beta1.ConfigElement{
			Type: tmv1beta1.ConfigTypeFile,
			Name: secret.Name,
			Path: kubeconfigPath,
			ValueFrom: &strconf.ConfigSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secret.Name},
					Key:                  "kubeconfig",
				},
			},
		}, config.LevelTestDefinition))
		return nil
	}
	if kubeconfig.Type == strconf.Config {
		*configs = append(*configs, config.NewElement(&tmv1beta1.ConfigElement{
			Type:      tmv1beta1.ConfigTypeFile,
			Name:      name,
			Path:      kubeconfigPath,
			ValueFrom: kubeconfig.Config(),
		}, config.LevelTestDefinition))
		return nil
	}
	return fmt.Errorf("undefined StringSecType %s", strconf.TypeToString(kubeconfig.Type))
}
