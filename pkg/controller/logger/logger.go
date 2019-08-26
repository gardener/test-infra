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

package logger

import (
	"flag"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var configFromFlags = Config{}

var developmentConfig = zap.Config{
	Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
	Development:       true,
	Encoding:          "console",
	DisableStacktrace: false,
	DisableCaller:     false,
	EncoderConfig:     zap.NewProductionEncoderConfig(),
	OutputPaths:       []string{"stderr"},
	ErrorOutputPaths:  []string{"stderr"},
}

var productionConfig = zap.Config{
	Level:             zap.NewAtomicLevelAt(zap.InfoLevel),
	Development:       false,
	DisableStacktrace: true,
	DisableCaller:     true,
	Encoding:          "json",
	EncoderConfig:     zap.NewProductionEncoderConfig(),
	OutputPaths:       []string{"stderr"},
	ErrorOutputPaths:  []string{"stderr"},
}

func New(config *Config) (logr.Logger, error) {
	if config == nil {
		config = &configFromFlags
	}
	zapCfg := determineZapConfig(config)

	level := int8(0 - config.Verbosity)
	zapCfg.Level = zap.NewAtomicLevelAt(zapcore.Level(level))

	zapLog, err := zapCfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}
	return zapr.NewLogger(zapLog), nil
}

func determineZapConfig(config *Config) zap.Config {
	var cfg zap.Config
	if config.Development {
		cfg = developmentConfig
	} else {
		cfg = productionConfig
	}

	cfg.DisableStacktrace = config.DisableStacktrace
	cfg.DisableCaller = config.DisableCaller

	return cfg
}

func InitFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}

	flag.BoolVar(&configFromFlags.Development, "dev", false, "enable development logging which result in console encoding, enabled stacktrace and enabled caller")
	flag.IntVar(&configFromFlags.Verbosity, "v", 1, "number for the log level verbosity")
	flag.BoolVar(&configFromFlags.DisableStacktrace, "disable-stacktrace", true, "disable the stacktrace of error logs")
	flag.BoolVar(&configFromFlags.DisableCaller, "disable-caller", true, "disable the caller of logs")
}
