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

package template

import (
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ = Describe("default templates", func() {

	It("should render the basic chart with all its necessary parameters", func() {
		params := &Parameters{
			GardenKubeconfigPath:    gardenerKubeconfig,
			DefaultTestrunChartPath: filepath.Join(defaultTestdataDir, "basic"),
			ComponentDescriptorPath: componentDescriptorPath,
		}
		runs, err := RenderTestruns(log.NullLogger{}, params, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(runs.GetTestruns()).To(HaveLen(1))
	})

	It("should render additional values to the chart", func() {
		params := &Parameters{
			GardenKubeconfigPath:    gardenerKubeconfig,
			DefaultTestrunChartPath: filepath.Join(defaultTestdataDir, "add-values"),
			ComponentDescriptorPath: componentDescriptorPath,
			SetValues:               []string{"addValue1=test,addValue2=test2"},
		}
		_, err := RenderTestruns(log.NullLogger{}, params, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should render multiple additional values to the chart", func() {
		params := &Parameters{
			GardenKubeconfigPath:    gardenerKubeconfig,
			DefaultTestrunChartPath: filepath.Join(defaultTestdataDir, "add-values"),
			ComponentDescriptorPath: componentDescriptorPath,
			SetValues:               []string{"addValue1=test", "addValue2=test2"},
		}
		_, err := RenderTestruns(log.NullLogger{}, params, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should add landscape and component descriptor as metadata", func() {
		params := &Parameters{
			GardenKubeconfigPath:    gardenerKubeconfig,
			DefaultTestrunChartPath: filepath.Join(defaultTestdataDir, "basic"),
			Landscape:               "test-landscape",
			ComponentDescriptorPath: componentDescriptorPath,
		}
		runs, err := RenderTestruns(log.NullLogger{}, params, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(runs.GetTestruns()).To(HaveLen(1))

		Expect(runs[0].Metadata).ToNot(BeNil())
		Expect(runs[0].Metadata.Landscape).To(Equal("test-landscape"))
		Expect(runs[0].Metadata.ComponentDescriptor).To(Equal(map[string]componentdescriptor.ComponentJSON{
			"github.com/gardener/gardener": {
				Version: "0.30.0",
			},
		}))
	})

})
