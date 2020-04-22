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

package util

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
	"strings"
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
