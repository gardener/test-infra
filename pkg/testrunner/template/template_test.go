// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"context"
	"path/filepath"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
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
