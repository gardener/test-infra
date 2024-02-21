// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

func RawJSONPath(content []byte, path string, dest interface{}) ([]byte, error) {
	subPaths := strings.Split(path, ".")
	if len(subPaths) == 0 {
		return content, nil
	}
	data, err := getRawJSONPath("", content, subPaths)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, dest); err != nil {
		return nil, err
	}
	return data, nil
}

func getRawJSONPath(identifier string, content []byte, path []string) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := yaml.Unmarshal(content, &obj); err != nil {
		return nil, err
	}

	subContent, ok := obj[path[0]]
	if !ok {
		return nil, fmt.Errorf("no object at %s.%s", identifier, path[0])
	}

	if len(path) == 1 {
		return subContent, nil
	}

	return getRawJSONPath(fmt.Sprintf("%s.%s", identifier, path[0]), subContent, path[1:])
}

func JSONPath(data interface{}, path string) (interface{}, error) {
	paths := strings.Split(path, ".")
	return getJSONPath(".", data, paths)
}

func getJSONPath(identifier string, data interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return nil, errors.New("path has to be defined")
	}
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil, errors.New("unable to cast to map")
	}

	elem, ok := m[path[0]]
	if !ok {
		return nil, errors.Errorf("unable to find path %s.%s", identifier, path[0])
	}

	if len(path) == 1 {
		return elem, nil
	}
	return getJSONPath(fmt.Sprintf("%s.%s", identifier, path[0]), elem, path[1:])
}
