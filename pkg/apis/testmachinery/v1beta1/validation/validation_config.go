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

package validation

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/gardener/test-infra/pkg/util/strconf"

	"k8s.io/apimachinery/pkg/util/validation"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// ValidateConfigList validates a list of configurations.
func ValidateConfigList(fldPath *field.Path, configs []tmv1beta1.ConfigElement) field.ErrorList {
	var allErrs field.ErrorList
	for i, config := range configs {
		iPath := fldPath.Index(i)
		allErrs = append(allErrs, ValidateConfig(iPath, config)...)
	}
	return allErrs
}

// ValidateConfig validates a testrun config element.
func ValidateConfig(fldPath *field.Path, config tmv1beta1.ConfigElement) field.ErrorList {
	var allErrs field.ErrorList
	if config.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("name"), "Required value"))
	}

	// configmaps should either have a value or a value from defined
	if len(config.Value) == 0 && config.ValueFrom == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("value/valueFrom"), "A config must consist of a value or a reference to a value"))
	}

	// if a valuefrom is defined then a configmap or a secret reference should be defined
	if config.ValueFrom != nil {
		allErrs = append(allErrs, strconf.Validate(fldPath.Child("valueFrom"), config.ValueFrom)...)
	}

	if config.Type != tmv1beta1.ConfigTypeEnv && config.Type != tmv1beta1.ConfigTypeFile {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("type"), config.Type, "unknown config type"))
		return allErrs
	}

	if config.Type == tmv1beta1.ConfigTypeEnv {
		if errs := validation.IsEnvVarName(config.Name); len(errs) != 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), config.Name, strings.Join(errs, ":")))
		}
		if errs := validation.IsCIdentifier(config.Name); len(errs) != 0 {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("name"), config.Name, strings.Join(errs, ":")))
		}
	}

	if config.Type == tmv1beta1.ConfigTypeFile {
		if config.Path == "" {
			allErrs = append(allErrs, field.Required(fldPath.Child("path"), fmt.Sprintf("path is required for configtype %q", tmv1beta1.ConfigTypeFile)))
		}
		// check if value is base64 encoded
		if config.Value != "" {
			if _, err := base64.StdEncoding.DecodeString(config.Value); err != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("value"), config.Value, "Value must be base64 encoded"))
			}
		}
	}
	return allErrs
}

// ValidateKubeconfigs validates all testrun kubeconfigs
func ValidateKubeconfigs(fldPath *field.Path, kubeconfigs tmv1beta1.TestrunKubeconfigs) field.ErrorList {
	var allErrs field.ErrorList
	// validate kubeconfigs
	k := reflect.ValueOf(kubeconfigs)
	typeOfK := k.Type()
	for i := 0; i < k.NumField(); i++ {
		allErrs = append(allErrs, ValidateKubeconfig(fldPath.Child("strconf").Child(typeOfK.Field(i).Name), k.Field(i).Interface().(*strconf.StringOrConfig))...)
	}
	return allErrs
}

// ValidateKubeconfig validates a kubeconfig definition.
func ValidateKubeconfig(fldPath *field.Path, kubeconfig *strconf.StringOrConfig) field.ErrorList {
	var allErrs field.ErrorList
	if kubeconfig == nil {
		return allErrs
	}

	switch kubeconfig.Type {
	case strconf.String:
		kubeconfig, err := base64.StdEncoding.DecodeString(kubeconfig.String())
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath, "", fmt.Sprintf("Cannot decode: %s", err.Error())))
			return allErrs
		}
		_, err = clientcmd.Load(kubeconfig)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(fldPath, "", fmt.Sprintf("Cannot build config: %s", err.Error())))
			return allErrs
		}
	case strconf.Config:
		allErrs = append(allErrs, strconf.Validate(fldPath, kubeconfig.Config())...)
	default:
		allErrs = append(allErrs, field.Invalid(fldPath.Child("type"), strconf.TypeToString(kubeconfig.Type), "Undefined StringSecType"))
	}
	return allErrs
}
