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

package viper

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

// Usage returns the help output for whole config
func (h *viperHelper) Usage() string {
	configMap := make(map[string]interface{})
	for _, f := range h.pflags {
		keys := strings.Split(GetConfigKey(f), ".")
		if err := createOrUpdateSubConfig(configMap, keys, f.Usage); err != nil {
			return ""
		}
	}

	dat, err := yaml.Marshal(configMap)
	if err != nil {
		return ""
	}
	return string(dat)
}

func GetConfigKey(flag *pflag.Flag) string {
	if flag.Annotations != nil {
		if key, ok := flag.Annotations[KeyAnnotation]; ok && len(key) != 0 {
			return key[0]
		}
	}
	return flag.Name
}

func createOrUpdateSubConfig(root map[string]interface{}, path []string, value string) error {
	if _, ok := root[path[0]]; !ok {
		root[path[0]] = createSubConfig(path[1:], value)
		return nil
	}
	sub, ok := root[path[0]].(map[string]interface{})
	if !ok {
		return errors.Errorf("unbale to add value to non map for path %s", strings.Join(path, "."))
	}

	return createOrUpdateSubConfig(sub, path[1:], value)
}

func createSubConfig(path []string, value string) interface{} {
	if len(path) == 0 {
		return value
	}
	return map[string]interface{}{path[0]: createSubConfig(path[1:], value)}
}

// AddCustomConfigForFlag sets a custom configuration key for the given flag
func AddCustomConfigForFlag(f *pflag.Flag, key string) {
	if f.Annotations == nil {
		f.Annotations = map[string][]string{}
	}
	f.Annotations[KeyAnnotation] = []string{key}
}
