// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
)

// NewElement creates a new config element object.
func NewElement(config *tmv1beta1.ConfigElement, level Level) *Element {
	name := fmt.Sprintf("%s-%s", config.Type, util.RandomString(5))
	return &Element{config, level, name}
}

// New creates new config elements.
func New(configs []tmv1beta1.ConfigElement, level Level) []*Element {
	var newConfigs []*Element
	for _, config := range configs {
		c := config
		newConfigs = append(newConfigs, NewElement(&c, level))
	}
	return newConfigs
}

// Name returns the config's unique name which is compatible to argo's parameter name conventions
func (c *Element) Name() string {
	return c.name
}

// Volume returns the pod volume for the current config if the ConfigType is file and
// a valuefrom is specified.
func (c *Element) Volume() (*corev1.Volume, error) {
	if c.Info.Type != tmv1beta1.ConfigTypeFile {
		return nil, fmt.Errorf("no volume for ConfigType 'file'")
	}
	if c.Info.Value != "" {
		return nil, fmt.Errorf("no volume for config with value")
	}
	volume := &corev1.Volume{
		Name: c.Name(),
	}

	if c.Info.ValueFrom.SecretKeyRef != nil {
		volume.Secret = &corev1.SecretVolumeSource{
			SecretName: c.Info.ValueFrom.SecretKeyRef.Name,
			Items: []corev1.KeyToPath{
				{
					Key:  c.Info.ValueFrom.SecretKeyRef.Key,
					Path: path.Base(c.Info.Path),
				},
			},
		}
	}
	if c.Info.ValueFrom.ConfigMapKeyRef != nil {
		volume.ConfigMap = &corev1.ConfigMapVolumeSource{
			LocalObjectReference: c.Info.ValueFrom.ConfigMapKeyRef.LocalObjectReference,
			Items: []corev1.KeyToPath{
				{
					Key:  c.Info.ValueFrom.ConfigMapKeyRef.Key,
					Path: path.Base(c.Info.Path),
				},
			},
		}
	}

	return volume, nil
}
