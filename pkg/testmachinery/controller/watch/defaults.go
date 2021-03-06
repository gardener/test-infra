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
