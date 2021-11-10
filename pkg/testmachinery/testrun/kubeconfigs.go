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
	"context"
	"encoding/base64"
	"fmt"
	"path"

	"github.com/gardener/test-infra/pkg/testmachinery/testflow/node"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	"github.com/gardener/test-infra/pkg/util/strconf"
)

const (
	hostKubeconfig     = "host"
	gardenerKubeconfig = "gardener"
	seedKubeconfig     = "seed"
	shootKubeconfig    = "shoot"
)

// ParseKubeconfigs parses the kubeconfigs defined in the testrun and returns respective configs and k8s secrets.
func ParseKubeconfigs(ctx context.Context, reader client.Reader, tr *tmv1beta1.Testrun) ([]*config.Element, []client.Object, map[string]*node.ProjectedTokenMount, error) {

	parsedKubeconfigs := make(map[string]*clientcmdv1.Config)
	configs := make([]*config.Element, 0)
	secrets := make([]client.Object, 0)
	if tr.Spec.Kubeconfigs.Host != nil {
		if err := addKubeconfig(ctx, reader, &configs, &secrets, parsedKubeconfigs, tr, hostKubeconfig, tr.Spec.Kubeconfigs.Host); err != nil {
			return nil, nil, nil, err
		}
	}
	if tr.Spec.Kubeconfigs.Gardener != nil {
		if err := addKubeconfig(ctx, reader, &configs, &secrets, parsedKubeconfigs, tr, gardenerKubeconfig, tr.Spec.Kubeconfigs.Gardener); err != nil {
			return nil, nil, nil, err
		}
	}
	if tr.Spec.Kubeconfigs.Seed != nil {
		if err := addKubeconfig(ctx, reader, &configs, &secrets, parsedKubeconfigs, tr, seedKubeconfig, tr.Spec.Kubeconfigs.Seed); err != nil {
			return nil, nil, nil, err
		}
	}
	if tr.Spec.Kubeconfigs.Shoot != nil {
		if err := addKubeconfig(ctx, reader, &configs, &secrets, parsedKubeconfigs, tr, shootKubeconfig, tr.Spec.Kubeconfigs.Shoot); err != nil {
			return nil, nil, nil, err
		}
	}

	projectedTokenMounts, err := processTokenFileConfigs(parsedKubeconfigs, tr.Name, tr.Namespace)
	if err != nil {
		return nil, nil, nil, err
	}

	return configs, secrets, projectedTokenMounts, nil
}

func addKubeconfig(ctx context.Context, reader client.Reader, configs *[]*config.Element, secrets *[]client.Object, parsedKubeconfigs map[string]*clientcmdv1.Config, tr *tmv1beta1.Testrun, name string, kubeconfig *strconf.StringOrConfig) error {
	var parsedKubeconfig clientcmdv1.Config
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

		err = yaml.Unmarshal(rawKubeconfig, &parsedKubeconfig)
		if err != nil {
			return err
		}
		parsedKubeconfigs[name] = &parsedKubeconfig

		return nil
	}
	if kubeconfig.Type == strconf.Config {
		*configs = append(*configs, config.NewElement(&tmv1beta1.ConfigElement{
			Type:      tmv1beta1.ConfigTypeFile,
			Name:      name,
			Path:      kubeconfigPath,
			ValueFrom: kubeconfig.Config(),
		}, config.LevelTestDefinition))

		if kubeconfig.Config().SecretKeyRef != nil {
			var kubeconfigSecret corev1.Secret

			objKey := client.ObjectKey{
				Namespace: tr.Namespace,
				Name:      kubeconfig.Config().SecretKeyRef.Name,
			}

			err := reader.Get(ctx, objKey, &kubeconfigSecret)
			if err != nil {
				return err
			}

			err = yaml.Unmarshal(kubeconfigSecret.Data[kubeconfig.Config().SecretKeyRef.Key], &parsedKubeconfig)
			if err != nil {
				return err
			}
			parsedKubeconfigs[name] = &parsedKubeconfig
		}
		return nil
	}
	return fmt.Errorf("undefined StringSecType %s", strconf.TypeToString(kubeconfig.Type))
}

// processTokenFileConfigs checks all given kubeconfig for tokenFiles, determines the allowed audience and check for conflicts in mountPaths
func processTokenFileConfigs(kubeconfigs map[string]*clientcmdv1.Config, testrunName, namespace string) (map[string]*node.ProjectedTokenMount, error) {
	landscapeMappings := testmachinery.GetLandscapeMappings()
	tokenMounts := make(map[string]*node.ProjectedTokenMount)

	if len(kubeconfigs) == 0 {
		return tokenMounts, nil
	}

	for name, kubeconfig := range kubeconfigs {

		if kubeconfig == nil {
			return nil, fmt.Errorf("kubeconfig Key for %s set but no actual kubeconfig was supplied in testrun", name)
		}

		if kubeconfig.CurrentContext == "" {
			return nil, fmt.Errorf("cannot process kubeconfig for %s due to missing currentContext field", name)
		}

		context := clientcmdv1.Context{}
		tokenMount := node.ProjectedTokenMount{}

		for _, c := range kubeconfig.Contexts {
			if c.Name == kubeconfig.CurrentContext {
				context = c.Context
			}
		}

		for _, auth := range kubeconfig.AuthInfos {
			if auth.Name == context.AuthInfo {
				if auth.AuthInfo.TokenFile != "" {
					dir, file := path.Split(auth.AuthInfo.TokenFile)
					tokenMount.MountPath = dir
					tokenMount.Name = file
				}
			}
		}

		if tokenMount.Name == "" || tokenMount.MountPath == "" {
			continue
		}

		for _, lm := range landscapeMappings {
			if lm.Namespace == namespace {
				if name == shootKubeconfig && !lm.AllowUntrustedUsage {
					return nil, fmt.Errorf("untrusted usage of tokenFile for kubeconfig %s is not allowed in landscapeMapping", name)
				}
				for _, c := range kubeconfig.Clusters {
					if c.Name == context.Cluster {
						if c.Cluster.Server == lm.ApiServerUrl {
							tokenMount.Audience = lm.Audience
							tokenMount.ExpirationSeconds = lm.ExpirationSeconds
						}
					}
				}
			}
		}

		if tokenMount.Audience == "" {
			return nil, fmt.Errorf("testrun wants to use a tokenFile for kubeconfig %s, but no matching landsacpeMapping was found", name)
		}

		for existingName, existingTokenMount := range tokenMounts {
			if existingTokenMount.MountPath == tokenMount.MountPath && existingTokenMount.Name == tokenMount.Name {
				return nil, fmt.Errorf("kubeconfigs for %s and %s both point to the exact same tokenFile location. Use a unique location per kubeconfig", existingName, name)
			}
		}

		tokenMounts[name] = &tokenMount

	}
	return tokenMounts, nil
}
