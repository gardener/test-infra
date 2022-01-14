// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gardener

import (
	"fmt"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardencorescheme "github.com/gardener/gardener/pkg/client/core/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	corescheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	// GardenScheme is the scheme used in the Garden cluster.
	GardenScheme = runtime.NewScheme()
	// SeedScheme is the scheme used in the Seed cluster.
	SeedScheme = runtime.NewScheme()
	// ShootScheme is the scheme used in the Shoot cluster.
	ShootScheme = runtime.NewScheme()
)

func init() {
	gardenSchemeBuilder := runtime.NewSchemeBuilder(
		corescheme.AddToScheme,
		gardencorescheme.AddToScheme,
	)
	utilruntime.Must(gardenSchemeBuilder.AddToScheme(GardenScheme))

	seedSchemeBuilder := runtime.NewSchemeBuilder(
		corescheme.AddToScheme,
		extensionsv1alpha1.AddToScheme,
	)
	utilruntime.Must(seedSchemeBuilder.AddToScheme(SeedScheme))

	shootSchemeBuilder := runtime.NewSchemeBuilder(
		corescheme.AddToScheme,
	)
	utilruntime.Must(shootSchemeBuilder.AddToScheme(ShootScheme))
}

// ShootCreationCompleted checks if a shoot is successfully reconciled. In case it is not, it also returns a descriptive message stating the reason.
func ShootCreationCompleted(newShoot *gardencorev1beta1.Shoot) (bool, string) {
	if newShoot.Generation != newShoot.Status.ObservedGeneration {
		return false, "shoot generation did not equal observed generation"
	}
	if len(newShoot.Status.Conditions) == 0 && newShoot.Status.LastOperation == nil {
		return false, "no conditions and last operation present yet"
	}

	for _, condition := range newShoot.Status.Conditions {
		if condition.Status != gardencorev1beta1.ConditionTrue {
			return false, fmt.Sprintf("condition type %s is not true yet, had message %s with reason %s", condition.Type, condition.Message, condition.Reason)
		}
	}

	if newShoot.Status.LastOperation != nil {
		if newShoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeCreate ||
			newShoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeReconcile ||
			newShoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeRestore {
			if newShoot.Status.LastOperation.State != gardencorev1beta1.LastOperationStateSucceeded {
				return false, "last operation type was create, reconcile or restore but state was not succeeded"
			}
		} else if newShoot.Status.LastOperation.Type == gardencorev1beta1.LastOperationTypeMigrate {
			return false, "last operation type was migrate, the migration process is not finished yet"
		}
	}

	return true, ""
}
