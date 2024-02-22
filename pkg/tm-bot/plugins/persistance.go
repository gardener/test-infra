// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"

	"github.com/ghodss/yaml"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kubernetesPersistence implements the Plugins persitence interface to store states in a configmap inside the cluster
type kubernetesPersistence struct {
	cm client.ObjectKey

	k8sClient client.Client
}

func NewKubernetesPersistence(k8sClient client.Client, name, namespace string) (Persistence, error) {
	ctx := context.Background()
	defer ctx.Done()

	cmKey := client.ObjectKey{Name: name, Namespace: namespace}
	cm := &corev1.ConfigMap{}
	if err := k8sClient.Get(ctx, cmKey, cm); err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}

		cm.Name = name
		cm.Namespace = namespace
		if err := k8sClient.Create(ctx, cm); err != nil {
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

	if err := p.k8sClient.Update(context.TODO(), cm); err != nil {
		return err
	}

	return nil
}
func (p *kubernetesPersistence) Load() (map[string]map[string]*State, error) {
	cm := &corev1.ConfigMap{}
	if err := p.k8sClient.Get(context.TODO(), p.cm, cm); err != nil {
		return nil, err
	}

	states := map[string]map[string]*State{}
	if err := yaml.Unmarshal([]byte(cm.Data["Plugins"]), &states); err != nil {
		return nil, err
	}

	return states, nil
}
