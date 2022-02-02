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
	"errors"
	"fmt"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"k8s.io/apimachinery/pkg/labels"
)

func (c *controller) determineShootInternalEndpoint(shoot *gardencorev1beta1.Shoot) (string, error) {
	projects, err := c.projects.Lister().List(labels.Everything())
	if err != nil {
		return "", err
	}

	var projectName string
	for _, project := range projects {
		if project == nil || project.Spec.Namespace == nil {
			continue
		}
		if *project.Spec.Namespace == shoot.ObjectMeta.Namespace {
			projectName = project.Name
			break
		}
	}

	if projectName == "" {
		return "", fmt.Errorf("Could not determine project name for Shoot %s/%s", shoot.ObjectMeta.Namespace, shoot.ObjectMeta.Name)
	}
	return fmt.Sprintf("https://api.%s.%s.%s", shoot.ObjectMeta.Name, strings.Replace(projectName, "garden-", "", -1), c.domain), nil
}

func (c *controller) fetchInternalDomain() error {
	selector := map[string]string{"gardener.cloud/role": "internal-domain"}
	secrets, err := c.secrets.Lister().Secrets("garden").List(labels.SelectorFromSet(labels.Set(selector)))
	if err != nil {
		return err
	}
	if len(secrets) != 1 {
		return errors.New("found no or multiple internal domain secrets")
	}
	if domain, exists := secrets[0].Annotations["dns.gardener.cloud/domain"]; exists {
		c.domain = domain
		return nil
	}
	return errors.New("could not fetch the internal domain")
}
