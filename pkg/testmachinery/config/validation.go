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
)

// Validate validates a testrun config element.
func Validate(identifier string, config tmv1beta1.ConfigElement) error {
	if config.Name == "" {
		return fmt.Errorf("%s.name: Required value", identifier)
	}
	if config.Type != tmv1beta1.ConfigTypeEnv {
		return fmt.Errorf("%s.type: Unsupported value: the specified type \"%s\" is unknown in config element %s", identifier, config.Type, config.Name)
	}
	return nil
}
