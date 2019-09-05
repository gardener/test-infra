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

package cmdutil

import (
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var ViperHelper *viperHelper

type viperHelper struct {
	pflags map[string]*flag.Flag
}

// NewViperHelper creates a new global ViperHelper instance.
func NewViperHelper() {
	ViperHelper = &viperHelper{pflags: map[string]*flag.Flag{}}
}

// BindPFlag binds a pflag to viper and stores a internal reference
func (h *viperHelper) BindPFlag(key string, f *flag.Flag) {
	h.pflags[key] = f
	_ = viper.BindPFlag(key, f)
}

// ApplyConfig writes viper flags back to the originated pflag variable pointer.
func (h *viperHelper) ApplyConfig() {
	for key, f := range h.pflags {
		_ = f.Value.Set(viper.GetString(key))
	}
}
