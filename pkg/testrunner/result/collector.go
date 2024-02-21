// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package result

import (
	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/pkg/testrunner"
)

func New(log logr.Logger, config Config, kubeconfig string) (*Collector, error) {
	collector := &Collector{
		log:            log,
		config:         config,
		kubeconfigPath: kubeconfig,
		RunExecCh:      make(chan *testrunner.Run),
	}

	return collector, nil
}
