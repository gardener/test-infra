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

package util

import (
	"encoding/json"
	"fmt"
	"sigs.k8s.io/yaml"
	"strings"
)

func JSONPath(content []byte, path string, dest interface{}) ([]byte, error) {
	subPaths := strings.Split(path, ".")
	if len(subPaths) == 0 {
		return content, nil
	}
	data, err := getJSONPath("", content, subPaths)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, dest); err != nil {
		return nil, err
	}
	return data, nil
}

func getJSONPath(identifier string, content []byte, path []string) ([]byte, error) {
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

	return getJSONPath(fmt.Sprintf("%s.%s", identifier, path[0]), subContent, path[1:])
}
