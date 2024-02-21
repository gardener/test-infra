// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package framework

import (
	"fmt"
	"os"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/gardener/test-infra/pkg/util"
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
