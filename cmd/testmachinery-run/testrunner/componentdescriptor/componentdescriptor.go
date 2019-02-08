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
package componentdescriptor

import (
	yaml "gopkg.in/yaml.v2"
)

// GetComponents returns a list of all git/component dependencies.
func GetComponents(content []byte) ([]*Component, error) {

	components := components{
		components:   make([]*Component, 0, 0),
		componentSet: make(map[Component]bool),
	}

	componentDescriptor, err := parse(content)
	if err != nil {
		return nil, err
	}

	for _, component := range componentDescriptor.Components {
		components.add(component.Dependencies.Components)
	}

	return components.components, nil
}

func (c *components) add(newComponents []Component) {
	for _, item := range newComponents {
		component := item
		if !c.has(component) {
			c.components = append(c.components, &component)
			c.componentSet[component] = true
		}
	}
}

func (c *components) has(newComponent Component) bool {
	_, ok := c.componentSet[newComponent]
	return ok
}

func parse(content []byte) (*descriptor, error) {
	var d = &descriptor{}
	err := yaml.Unmarshal(content, d)
	if err != nil {
		return nil, err
	}
	return d, nil
}
