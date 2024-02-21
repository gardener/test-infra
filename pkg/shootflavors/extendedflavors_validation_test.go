// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shootflavors

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/gardener/test-infra/pkg/common"
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
