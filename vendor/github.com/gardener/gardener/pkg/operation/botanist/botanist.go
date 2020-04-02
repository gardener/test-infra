// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package botanist

import (
	"context"
	"fmt"
	"sort"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"
	"github.com/gardener/gardener/pkg/operation"
	"github.com/gardener/gardener/pkg/operation/common"
	shootpkg "github.com/gardener/gardener/pkg/operation/shoot"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// New takes an operation object <o> and creates a new Botanist object. It checks whether the given Shoot DNS
// domain is covered by a default domain, and if so, it sets the <DefaultDomainSecret> attribute on the Botanist
// object.
func New(o *operation.Operation) (*Botanist, error) {
	b := &Botanist{
		Operation: o,
	}

	// Determine all default domain secrets and check whether the used Shoot domain matches a default domain.
	if o.Shoot != nil && o.Shoot.Info.Spec.DNS != nil && o.Shoot.Info.Spec.DNS.Domain != nil {
		var (
			prefix            = fmt.Sprintf("%s-", common.GardenRoleDefaultDomain)
			defaultDomainKeys = o.GetSecretKeysOfRole(common.GardenRoleDefaultDomain)
		)
		sort.Slice(defaultDomainKeys, func(i, j int) bool { return len(defaultDomainKeys[i]) >= len(defaultDomainKeys[j]) })
		for _, key := range defaultDomainKeys {
			defaultDomain := strings.SplitAfter(key, prefix)[1]
			if strings.HasSuffix(*(o.Shoot.Info.Spec.DNS.Domain), defaultDomain) {
				b.DefaultDomainSecret = b.Secrets[prefix+defaultDomain]
				break
			}
		}
	}

	if err := b.InitializeSeedClients(); err != nil {
		return nil, err
	}

	return b, nil
}

// RequiredExtensionsExist checks whether all required extensions needed for an shoot operation exist.
func (b *Botanist) RequiredExtensionsExist(ctx context.Context) error {
	controllerRegistrationList := &gardencorev1beta1.ControllerRegistrationList{}
	if err := b.K8sGardenClient.Client().List(ctx, controllerRegistrationList); err != nil {
		return err
	}

	controllerInstallationList := &gardencorev1beta1.ControllerInstallationList{}
	if err := b.K8sGardenClient.Client().List(ctx, controllerInstallationList); err != nil {
		return err
	}

	var controllerRegistrations []*gardencorev1beta1.ControllerRegistration
	for _, controllerRegistration := range controllerRegistrationList.Items {
		controllerRegistrations = append(controllerRegistrations, controllerRegistration.DeepCopy())
	}

	requiredExtensions := shootpkg.ComputeRequiredExtensions(b.Shoot.Info, b.Seed.Info, controllerRegistrations, b.Garden.InternalDomain, b.Shoot.ExternalDomain)

	for _, controllerInstallation := range controllerInstallationList.Items {
		if controllerInstallation.Spec.SeedRef.Name != b.Seed.Info.Name {
			continue
		}

		controllerRegistration := &gardencorev1beta1.ControllerRegistration{}
		if err := b.K8sGardenClient.Client().Get(ctx, client.ObjectKey{Name: controllerInstallation.Spec.RegistrationRef.Name}, controllerRegistration); err != nil {
			return err
		}

		for _, kindType := range requiredExtensions.UnsortedList() {
			split := strings.Split(kindType, "/")
			if len(split) != 2 {
				return fmt.Errorf("unexpected required extension: %q", kindType)
			}
			extensionKind, extensionType := split[0], split[1]

			if helper.IsResourceSupported(controllerRegistration.Spec.Resources, extensionKind, extensionType) && helper.IsControllerInstallationSuccessful(controllerInstallation) {
				requiredExtensions.Delete(kindType)
			}
		}
	}

	if len(requiredExtensions) > 0 {
		return fmt.Errorf("extension controllers missing or unready: %+v", requiredExtensions)
	}

	return nil
}
