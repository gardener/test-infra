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

package framework

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"os"
)

// ValidateConfig validates a framework configuration
func ValidateConfig(config *Config) error {
	if config == nil {
		return errors.New("no config is defined")
	}

	var res *multierror.Error

	if _, err := os.Stat(config.TmKubeconfigPath); err != nil {
		if os.IsNotExist(err) {
			res = multierror.Append(res, fmt.Errorf("file %s does not exist", config.TmKubeconfigPath))
		} else {
			res = multierror.Append(res, err)
		}
	}

	if len(config.TmNamespace) == 0 {
		res = multierror.Append(res, errors.New("no Test Machinery namespace is defined"))
	}

	return util.ReturnMultiError(res)
}
