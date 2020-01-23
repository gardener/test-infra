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

package shootflavors

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("extended flavor validation test", func() {
	var (
		flavors []*common.ExtendedShootFlavor
	)

	BeforeEach(func() {
		flavors = []*common.ExtendedShootFlavor{{
			ShootFlavor: common.ShootFlavor{
				KubernetesVersions: common.ShootKubernetesVersionFlavor{
					Versions: &[]gardencorev1beta1.ExpirableVersion{
						{
							Version: "1.15",
						},
					},
				},
				Workers: []common.ShootWorkerFlavor{
					{
						WorkerPools: []gardencorev1beta1.Worker{{Name: "wp1"}},
					},
				},
			},
			ExtendedConfiguration: common.ExtendedConfiguration{
				ProjectName:      "test",
				CloudprofileName: "test-prov",
				SecretBinding:    "sb-test",
				Region:           "test-region",
				Zone:             "test-zone",
			},
		}}
	})

	It("should not throw an error", func() {
		err := ValidateExtendedFlavor("", flavors[0])
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("validate errors", func(flavors func() []*common.ExtendedShootFlavor) {
		err := ValidateExtendedFlavor("", flavors()[0])
		Expect(err).To(HaveOccurred())
	},
		Entry("no workers", func() []*common.ExtendedShootFlavor {
			flavors[0].ShootFlavor.Workers = nil
			return flavors
		}),
		Entry("no kubernetes version", func() []*common.ExtendedShootFlavor {
			flavors[0].ShootFlavor.KubernetesVersions = common.ShootKubernetesVersionFlavor{}
			return flavors
		}),
		Entry("no cloudprofile", func() []*common.ExtendedShootFlavor {
			flavors[0].ExtendedConfiguration.CloudprofileName = ""
			return flavors
		}),
		Entry("no provider", func() []*common.ExtendedShootFlavor {
			flavors[0].ShootFlavor.Provider = ""
			return flavors
		}),
		Entry("no project name", func() []*common.ExtendedShootFlavor {
			flavors[0].ExtendedConfiguration.ProjectName = ""
			return flavors
		}),
		Entry("no secret binding", func() []*common.ExtendedShootFlavor {
			flavors[0].ExtendedConfiguration.SecretBinding = ""
			return flavors
		}),
		Entry("no region", func() []*common.ExtendedShootFlavor {
			flavors[0].ExtendedConfiguration.Region = ""
			return flavors
		}))
})
