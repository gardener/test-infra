// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testrunner

import (
	"fmt"

	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/pkg/util"
)

// ExecuteTestruns deploys it to a testmachinery cluster and waits for the testruns results
func ExecuteTestruns(log logr.Logger, config *Config, runs RunList, testrunNamePrefix string, notify ...chan *Run) error {
	log.V(3).Info(fmt.Sprintf("Config: %+v", util.PrettyPrintStruct(config)))

	return runs.Run(log.WithValues("namespace", config.Namespace), config, testrunNamePrefix, notify...)
}
