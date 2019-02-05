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
	"fmt"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
)

// newElement creates a new config element object.
func newElement(config *tmv1beta1.ConfigElement) *Element {
	name := fmt.Sprintf("%s-%s", config.Type, util.RandomString(3))
	return &Element{config, name}
}

// New creates new config elements.
func New(configs []tmv1beta1.ConfigElement) []*Element {
	var newConfigs []*Element
	for _, config := range configs {
		c := config
		newConfigs = append(newConfigs, newElement(&c))
	}
	return newConfigs
}

// Name returns the config's unique name which is compatible to argo's parameter name conventions
func (c *Element) Name() string {
	return c.name
}
