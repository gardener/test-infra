// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
