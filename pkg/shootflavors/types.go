// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package shootflavors

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/gardener/test-infra/pkg/common"
)

// Flavors represents the internal versions of a shoot flavor.
// Has be to be initiated by New
type Flavors struct {
	Info   []*common.ShootFlavor
	shoots []*common.Shoot

	usedKubernetesVersions map[common.CloudProvider]gardencorev1beta1.KubernetesSettings
	usedMachineImages      map[common.CloudProvider][]gardencorev1beta1.MachineImage
}

// Flavors represents the internal versions of a extended shoot flavor.
// Has be to be initiated by NewExtended
type ExtendedFlavors struct {
	Info   []*common.ExtendedShootFlavor
	shoots []*ExtendedFlavorInstance

	usedKubernetesVersions map[common.CloudProvider]gardencorev1beta1.KubernetesSettings
}

// ExtendedFlavorInstance defines a instance of a shoot flavor
type ExtendedFlavorInstance struct {
	shoot *common.ExtendedShoot
}
