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

package config

import (
	"encoding/base64"
	"fmt"
	"strings"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"k8s.io/apimachinery/pkg/util/validation"
)

// Validate validates a testrun config element.
func Validate(identifier string, config tmv1beta1.ConfigElement) error {
	if config.Name == "" {
		return fmt.Errorf("%s.name: Required value", identifier)
	}

	// configmaps should either have a value or a value from defined
	if config.Value == "" && config.ValueFrom == nil {
		return fmt.Errorf("%s.(value or valueFrom): Required value or valueFrom: A config must consist of a value or a reference to a value", identifier)
	}

	// if a valuefrom is defined then a configmap or a secret reference should be defined
	if config.ValueFrom != nil {
		if config.ValueFrom.ConfigMapKeyRef == nil && config.ValueFrom.SecretKeyRef == nil {
			return fmt.Errorf("%s.valueFrom.(configMapKeyRef or secretMapKeyRef): Required configMapKeyRef or secretMapKeyRef: Either a configmap ref or a secretmap ref have to be defined", identifier)
		}
		if config.ValueFrom.ConfigMapKeyRef != nil {
			if config.ValueFrom.ConfigMapKeyRef.Key == "" {
				return fmt.Errorf("%s.valueFrom.configMapKeyRef.key: Required value", identifier)
			}
			if config.ValueFrom.ConfigMapKeyRef.Name == "" {
				return fmt.Errorf("%s.valueFrom.configMapKeyRef.name: Required value", identifier)
			}
		}
		if config.ValueFrom.SecretKeyRef != nil {
			if config.ValueFrom.SecretKeyRef.Key == "" {
				return fmt.Errorf("%s.valueFrom.secretKeyRef.key: Required value", identifier)
			}
			if config.ValueFrom.SecretKeyRef.Name == "" {
				return fmt.Errorf("%s.valueFrom.secretKeyRef.name: Required value", identifier)
			}
		}
	}

	if config.Type == tmv1beta1.ConfigTypeEnv {
		if errs := validation.IsEnvVarName(config.Name); len(errs) != 0 {
			return fmt.Errorf("%s.name: Invalid value: %s", identifier, strings.Join(errs, ":"))
		}
		if errs := validation.IsCIdentifier(config.Name); len(errs) != 0 {
			return fmt.Errorf("%s.name: Invalid value: %s", identifier, strings.Join(errs, ":"))
		}
		return nil
	}

	if config.Type == tmv1beta1.ConfigTypeFile {
		if config.Path == "" {
			return fmt.Errorf("%s.path: Required value: path is required for configtype '%s'", identifier, tmv1beta1.ConfigTypeFile)
		}

		// check if value is base64 encoded
		if _, err := base64.StdEncoding.DecodeString(config.Value); err != nil {
			return fmt.Errorf("%s.value: Invalid value: Value must be base64 encoded", identifier)
		}

		return nil
	}
	return fmt.Errorf("%s.type: Unsupported value: the specified type \"%s\" is unknown in config element %s", identifier, config.Type, config.Name)
}
