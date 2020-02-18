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
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"time"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
)

// Config is an app config
type Config struct {
	KubeConfigPath     string
	KubeConfig         clientcmd.ClientConfig
	CheckIntervalInput string
	CheckInterval      time.Duration
	OutputDir          string
	DisableAnalyse     bool
	AnalyseFormat      string
	AnalyseOutput      string
	ShootNames         []string
	ShootsFilter       map[string]bool
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

	return nil
}

// ValidateAnalyse validates the input for the case of running only a data analysis.
func ValidateAnalyse(path, format string) error {
	info, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("no directory on path %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is expected be a directory", path)
	}

	measurementsDir := common.GetResultDir(path)
	info, err = os.Stat(measurementsDir)
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("no directory on path %s", measurementsDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is expected be a directory", measurementsDir)
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
