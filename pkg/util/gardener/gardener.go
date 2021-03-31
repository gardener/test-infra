// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gardener

import (
	"fmt"

	mrv1alpha1 "github.com/gardener/gardener-resource-manager/api/resources/v1alpha1"
	mrhelper "github.com/gardener/gardener-resource-manager/api/resources/v1alpha1/helper"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	gardencorescheme "github.com/gardener/gardener/pkg/client/core/clientset/versioned/scheme"
	gardenextensionsscheme "github.com/gardener/gardener/pkg/client/extensions/clientset/versioned/scheme"
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
		gardenextensionsscheme.AddToScheme,
		mrv1alpha1.AddToScheme,
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

// CheckManagedResource checks if all conditions of a ManagedResource ('ResourcesApplied' and 'ResourcesHealthy')
// are True and .status.observedGeneration matches the current .metadata.generation
func CheckManagedResource(mr *mrv1alpha1.ManagedResource) error {
	if err := CheckManagedResourceApplied(mr); err != nil {
		return err
	}
	if err := CheckManagedResourceHealthy(mr); err != nil {
		return err
	}

	return nil
}

// CheckManagedResourceApplied checks if the condition 'ResourcesApplied' of a ManagedResource
// is True and the .status.observedGeneration matches the current .metadata.generation
func CheckManagedResourceApplied(mr *mrv1alpha1.ManagedResource) error {
	status := mr.Status
	if status.ObservedGeneration != mr.GetGeneration() {
		return fmt.Errorf("observed generation of managed resource %s/%s outdated (%d/%d)", mr.GetNamespace(), mr.GetName(), status.ObservedGeneration, mr.GetGeneration())
	}

	conditionApplied := mrhelper.GetCondition(status.Conditions, mrv1alpha1.ResourcesApplied)

	if conditionApplied == nil {
		return fmt.Errorf("condition %s for managed resource %s/%s has not been reported yet", mrv1alpha1.ResourcesApplied, mr.GetNamespace(), mr.GetName())
	} else if conditionApplied.Status != mrv1alpha1.ConditionTrue {
		return fmt.Errorf("condition %s of managed resource %s/%s is %s: %s", mrv1alpha1.ResourcesApplied, mr.GetNamespace(), mr.GetName(), conditionApplied.Status, conditionApplied.Message)
	}

	return nil
}

// CheckManagedResourceHealthy checks if the condition 'ResourcesHealthy' of a ManagedResource is True
func CheckManagedResourceHealthy(mr *mrv1alpha1.ManagedResource) error {
	status := mr.Status
	conditionHealthy := mrhelper.GetCondition(status.Conditions, mrv1alpha1.ResourcesHealthy)

	if conditionHealthy == nil {
		return fmt.Errorf("condition %s for managed resource %s/%s has not been reported yet", mrv1alpha1.ResourcesHealthy, mr.GetNamespace(), mr.GetName())
	} else if conditionHealthy.Status != mrv1alpha1.ConditionTrue {
		return fmt.Errorf("condition %s of managed resource %s/%s is %s: %s", mrv1alpha1.ResourcesHealthy, mr.GetNamespace(), mr.GetName(), conditionHealthy.Status, conditionHealthy.Message)
	}

	return nil
}
