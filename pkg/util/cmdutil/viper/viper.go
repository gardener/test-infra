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

package viper

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	KeyAnnotation = "key"
)

type viperHelper struct {
	viper  *viper.Viper
	pflags map[string]*flag.Flag

	customConfigPath string
}

// NewViperHelper creates a new global ViperHelper instance.
func NewViperHelper(v *viper.Viper, name string, configPaths ...string) *viperHelper {
	if v == nil {
		v = viper.GetViper()
	}
	v.SetConfigName(name)

	vh := &viperHelper{
		viper:  v,
		pflags: map[string]*flag.Flag{},
	}
	for _, p := range configPaths {
		v.AddConfigPath(p)
	}
	return vh
}

// Add viper init flags
func (h *viperHelper) InitFlags(fs *flag.FlagSet) {
	if fs == nil {
		fs = flag.CommandLine
	}
	fs.StringVar(&h.customConfigPath, "custom-config", "", "Specify a custom config to the configuration file")
}

// BindPFlag binds a pflag to viper and stores a internal reference
func (h *viperHelper) BindPFlag(key string, f *flag.Flag) {
	AddCustomConfigForFlag(f, key)
	h.pflags[key] = f
	_ = h.viper.BindPFlag(key, f)
}

// BindPFlag binds a pflag to viper and stores a internal reference
func BindPFlag(key string, f *flag.Flag) {
	ViperHelper.BindPFlag(key, f)
}

// BindPFlagFromFlagSet sets a custom configuration key for the given flag name
func BindPFlagFromFlagSet(fs *flag.FlagSet, name, key string) {
	if f := fs.Lookup(name); f != nil {
		ViperHelper.BindPFlag(key, f)
	}
}

// BindPFlags binds all pflag of a flagset to viper and stores a internal reference
func (h *viperHelper) BindPFlags(fs *flag.FlagSet, keyPrefix string) {
	fs.VisitAll(func(f *flag.Flag) {
		key := GetConfigKey(f)
		if keyPrefix != "" {
			key = fmt.Sprintf("%s.%s", keyPrefix, key)
		}
		h.BindPFlag(key, f)
	})
}

// ReadInConfig will discover and load the configuration file from disk
// and key/value stores, searching in one of the defined paths.
func (h *viperHelper) ReadInConfig() error {
	if h.customConfigPath != "" {
		file, err := os.Open(h.customConfigPath)
		if err != nil {
			return errors.Wrapf(err, "unable to read file from %s", h.customConfigPath)
		}
		if err := h.viper.ReadConfig(file); err != nil {
			return err
		}
	} else {
		if err := h.viper.ReadInConfig(); err != nil {
			return err
		}
	}
	h.ApplyConfig()
	return nil
}

// ApplyConfig writes viper flags back to the originated pflag variable pointer.
func (h *viperHelper) ApplyConfig() {
	for key, f := range h.pflags {
		_ = f.Value.Set(h.viper.GetString(key))
	}
}

var ViperHelper *viperHelper = NewViperHelper(nil, "config", fmt.Sprintf("$HOME/.%s", os.Args[0]))

func SetViper(helper *viperHelper) {
	ViperHelper = helper
}

func InitFlags(fs *flag.FlagSet) {
	ViperHelper.InitFlags(fs)
}
