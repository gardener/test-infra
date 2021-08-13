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

package template

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gardener/component-cli/pkg/commands/constants"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testdataDir             string
	defaultTestdataDir      string
	shootTestdataDir        string
	gardenerKubeconfig      string
	componentCacheDir       string
	componentDescriptorPath string
)

func TestTestrunnerTemplate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testrunner Template Test Suite")
}

var _ = BeforeSuite(func() {
	wd, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	testdataDir, err = filepath.Abs(filepath.Join(wd, "testdata"))
	Expect(err).ToNot(HaveOccurred())
	componentCacheDir = testdataDir

	defaultTestdataDir = filepath.Join(testdataDir, "default")
	shootTestdataDir = filepath.Join(testdataDir, "shoot")

	gardenerKubeconfig = filepath.Join(testdataDir, "test-kubeconfig.yaml")
	componentDescriptorPath = filepath.Join(componentCacheDir, "registry.example/github.com/gardener/gardener-0.30.0")
	Expect(os.Setenv(constants.ComponentRepositoryCacheDirEnvVar, componentCacheDir))
})
