// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package watch

import (
	"time"

	"github.com/gardener/test-infra/pkg/testmachinery"
)

func applyDefaultOptions(opts *Options) *Options {
	if opts == nil {
		opts = &Options{}
	}

	if len(opts.InformerType) == 0 {
		opts.InformerType = CachedInformerType
	}

	if opts.Scheme == nil {
		opts.Scheme = testmachinery.TestMachineryScheme
	}

	if opts.SyncPeriod == nil {
		d := 10 * time.Minute
		opts.SyncPeriod = &d
	}
	if opts.PollInterval == nil {
		d := 1 * time.Minute
		opts.PollInterval = &d
	}
	return opts
}
