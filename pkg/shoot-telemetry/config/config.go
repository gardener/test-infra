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
	"github.com/pkg/errors"
	"os"
	"time"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	log "github.com/sirupsen/logrus"
)

// Config is an app config
type Config struct {
	KubeConfigPath     string
	CheckIntervalInput string
	CheckInterval      time.Duration
	OutputDir          string
	OutputFile         string
	DisableAnalyse     bool
	AnalyseFormat      string
	AnalyseOutput      string
	ShootName          string
	ShootNamespace     string
	LogLevel           string
}

// Validate returns an error if the Config is not valid.
func (c *Config) Validate() error {
	if c.KubeConfigPath != "" {
		_, err := os.Stat(c.KubeConfigPath)
		if err != nil && os.IsNotExist(err) {
			return fmt.Errorf("no kubeconfig of path %s", c.KubeConfigPath)
		}
	}

	if _, err := os.Stat(c.OutputDir); err != nil && os.IsNotExist(err) {
		fmt.Println(c.OutputDir)
		return fmt.Errorf("output directory does not exists: %s", c.OutputDir)
	}

	if err := validateOutputFormat(c.AnalyseFormat); err != nil {
		return err
	}

	if c.ShootName != "" && c.ShootNamespace == "" {
		return errors.New("project has to be defined if a shoot is targeted")
	}

	return nil
}

// ValidateAnalyse validates the input for the case of running only a data analysis.
func ValidateAnalyse(path, format string) error {
	info, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("no file on path %s", path)
	}
	if info.IsDir() {
		return fmt.Errorf("can't analyse a directory %s", path)
	}

	if err := validateOutputFormat(format); err != nil {
		return err
	}
	return nil
}

// validateOutputFormat validates if the passed format is one of the supported formats.
func validateOutputFormat(format string) error {
	if format == common.ReportOutputFormatText || format == common.ReportOutputFormatJSON {
		return nil
	}
	return fmt.Errorf("given format %s is invalid", format)
}

// SetupLogger configures the logger. The info log level will be ensured.
func SetupLogger(logLevel string) {
	// Format log output.
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
		DisableColors: true,
	})

	// Set the log level.
	switch logLevel {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.Infof("Log level %s can't be applied. Use info log level.", logLevel)
		log.SetLevel(log.InfoLevel)
	}
}
