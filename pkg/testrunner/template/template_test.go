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
	"context"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"path/filepath"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("default templates", func() {
	const (
		DEFAULT_TESTDATA_DIR = "./testdata/default"
		GARDENER_KUBECONFIG  = "./testdata/test-kubeconfig.yaml"

		COMPONENT_TESTDATA_PATH = "../componentdescriptor/testdata/"
		ROOT_COMPONENT          = "root-component.yaml"
		REPOSITORY              = "repositories/ocm-repo-ctf"
	)

	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	AfterEach(func() {
		ctx.Done()
	})

	It("should render the basic chart with all its necessary parameters", func() {
		params := &Parameters{
			GardenKubeconfigPath:    GARDENER_KUBECONFIG,
			DefaultTestrunChartPath: filepath.Join(DEFAULT_TESTDATA_DIR, "basic"),
			ComponentDescriptorPath: filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
			Repository:              filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
		}
		runs, err := RenderTestruns(ctx, logr.Discard(), params, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(runs.GetTestruns()).To(HaveLen(1))
		// expect 8 locations - the one predefined in the basic template and the locations extracted from the 7
		// components
		Expect(len(runs.GetTestruns()[0].Spec.LocationSets[0].Locations)).To(Equal(8))
	})

	It("should render additional values to the chart", func() {
		params := &Parameters{
			GardenKubeconfigPath:    GARDENER_KUBECONFIG,
			DefaultTestrunChartPath: filepath.Join(DEFAULT_TESTDATA_DIR, "add-values"),
			ComponentDescriptorPath: filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
			Repository:              filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
			SetValues:               []string{"addValue1=test,addValue2=test2"},
		}

		_, err := RenderTestruns(ctx, logr.Discard(), params, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should add landscape and component descriptor as metadata", func() {
		params := &Parameters{
			GardenKubeconfigPath:    GARDENER_KUBECONFIG,
			DefaultTestrunChartPath: filepath.Join(DEFAULT_TESTDATA_DIR, "basic"),
			Landscape:               "test-landscape",
			ComponentDescriptorPath: filepath.Join(COMPONENT_TESTDATA_PATH, ROOT_COMPONENT),
			Repository:              filepath.Join(COMPONENT_TESTDATA_PATH, REPOSITORY),
		}
		runs, err := RenderTestruns(ctx, logr.Discard(), params, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(runs.GetTestruns()).To(HaveLen(1))

		Expect(runs[0].Metadata).ToNot(BeNil())
		Expect(runs[0].Metadata.Landscape).To(Equal("test-landscape"))
		Expect(runs[0].Metadata.ComponentDescriptor).To(Equal(map[string]componentdescriptor.ComponentJSON{
			"github.com/component-3":       {Version: "v1.0.0"},
			"github.com/component-1-1":     {Version: "v1.0.0"},
			"github.com/component-2-1":     {Version: "v1.0.0"},
			"github.com/component-2-2":     {Version: "v1.0.0"},
			"github.com/gardener/gardener": {Version: "v1.0.0"},
			"github.com/component-1":       {Version: "v1.0.0"},
			"github.com/component-2":       {Version: "v1.0.0"},
		}))
	})
})
