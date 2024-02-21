// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"time"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gardener/test-infra/pkg/util"
)

var _ = Describe("gardener util", func() {

	Context("versions", func() {
		Context("filter", func() {
			It("should remove old 1.14 version and keep everything else", func() {
				versions := []gardencorev1beta1.ExpirableVersion{
					newExpirableVersion("1.13.5"),
					newExpirableVersion("1.14.3"),
					newExpirableVersion("1.14.4"),
					newExpirableVersion("1.15.0"),
				}

				Expect(util.FilterPatchVersions(versions)).To(ConsistOf(
					newExpirableVersion("1.13.5"),
					newExpirableVersion("1.14.4"),
					newExpirableVersion("1.15.0"),
				))
			})

			It("should remove old 1.14 and 1.13 version", func() {
				versions := []gardencorev1beta1.ExpirableVersion{
					newExpirableVersion("1.13.0"),
					newExpirableVersion("1.13.5"),
					newExpirableVersion("1.14.3"),
					newExpirableVersion("1.14.4"),
				}

				Expect(util.FilterPatchVersions(versions)).To(ConsistOf(
					newExpirableVersion("1.13.5"),
					newExpirableVersion("1.14.4"),
				))
			})

			Context("latest", func() {
				It("should return the latest version", func() {
					versions := []gardencorev1beta1.ExpirableVersion{
						newExpirableVersion("1.13.0"),
						newExpirableVersion("1.13.5"),
						newExpirableVersion("1.14.3"),
						newExpirableVersion("1.14.4"),
					}
					Expect(util.GetLatestVersion(versions)).To(Equal(newExpirableVersion("1.14.4")))
				})
			})

			It("should remove expired versions", func() {
				versions := []gardencorev1beta1.ExpirableVersion{
					newExpirableVersion("1.13.5"),
					newExpiredVersion("1.14.3"),
					newExpiredVersion("1.14.4"),
					newExpirableVersion("1.15.0"),
				}

				Expect(util.FilterExpiredVersions(versions)).To(ConsistOf(
					newExpirableVersion("1.13.5"),
					newExpirableVersion("1.15.0"),
				))
			})
		})

	})
})

func newExpirableVersion(v string) gardencorev1beta1.ExpirableVersion {
	return gardencorev1beta1.ExpirableVersion{Version: v}
}

func newExpiredVersion(v string) gardencorev1beta1.ExpirableVersion {
	pastTime := metav1.NewTime(time.Now().Add(-24 * time.Hour))
	return gardencorev1beta1.ExpirableVersion{Version: v, ExpirationDate: &pastTime}
}
