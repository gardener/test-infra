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
