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

package plugins

import (
	"context"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kubernetesPersistence implements the Plugins persitence interface to store states in a configmap inside the cluster
type kubernetesPersistence struct {
	cm client.ObjectKey

	k8sClient kubernetes.Interface
}

func NewKubernetesPersistence(k8sClient kubernetes.Interface, name, namespace string) (Persistence, error) {
	ctx := context.Background()
	defer ctx.Done()

	cmKey := client.ObjectKey{Name: name, Namespace: namespace}
	cm := &corev1.ConfigMap{}
	if err := k8sClient.Client().Get(ctx, cmKey, cm); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}

		cm.Name = name
		cm.Namespace = namespace
		if err := k8sClient.Client().Create(ctx, cm); err != nil {
			return nil, err
		}
	}

	return &kubernetesPersistence{
		k8sClient: k8sClient,
		cm:        cmKey,
	}, nil
}

func (p *kubernetesPersistence) Save(states map[string]map[string]*State) error {
	data, err := yaml.Marshal(states)
	if err != nil {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      p.cm.Name,
			Namespace: p.cm.Namespace,
		},
		Data: map[string]string{
			"Plugins": string(data),
		},
	}

	if err := p.k8sClient.Client().Update(context.TODO(), cm); err != nil {
		return err
	}

	return nil
}
func (p *kubernetesPersistence) Load() (map[string]map[string]*State, error) {
	cm := &corev1.ConfigMap{}
	if err := p.k8sClient.Client().Get(context.TODO(), p.cm, cm); err != nil {
		return nil, err
	}

	states := map[string]map[string]*State{}
	if err := yaml.Unmarshal([]byte(cm.Data["Plugins"]), &states); err != nil {
		return nil, err
	}

	return states, nil
}
