package gardener_telemetry_cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/test-infra/pkg/logger"
	telcommon "github.com/gardener/test-infra/pkg/shoot-telemetry/common"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/gardener/gardener/test/framework"
	"github.com/go-logr/logr"
)

func GetUnhealthyShoots(log logr.Logger, ctx context.Context, k8sClient kubernetes.Interface) (map[string]bool, error) {
	shoots := &gardencorev1beta1.ShootList{}
	err := k8sClient.Client().List(ctx, shoots)
	if err != nil {
		return nil, err
	}

	unhealthyShoots := make(map[string]bool)
	for _, shoot := range shoots.Items {

		// add shoots to unhelathy shoots if they are reconciled but a condition is unhealthy
		if shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateSucceeded {
			for _, condition := range shoot.Status.Conditions {
				if condition.Status != gardencorev1beta1.ConditionTrue {
					unhealthyShoots[telcommon.GetShootKeyFromShoot(&shoot)] = true
					log.V(5).Info("shoot is already unhealthy", "name", shoot.GetName(), "namespace", shoot.GetNamespace())
					continue
				}
			}
		}

		if shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateError || shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateFailed || shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateAborted {
			unhealthyShoots[telcommon.GetShootKeyFromShoot(&shoot)] = true
			log.V(5).Info("shoot is already unhealthy", "name", shoot.GetName(), "namespace", shoot.GetNamespace())
		} else if len(shoot.Status.LastErrors) != 0 {
			unhealthyShoots[telcommon.GetShootKeyFromShoot(&shoot)] = true
			log.V(5).Info("shoot is already unhealthy", "name", shoot.GetName(), "namespace", shoot.GetNamespace())
		}
	}

	log.Info(fmt.Sprintf("Found %d unhealthy shoots", len(unhealthyShoots)))
	return unhealthyShoots, nil
}

func WaitForGardenerUpdate(log logr.Logger, ctx context.Context, k8sClient kubernetes.Interface, newGardenerVersion string, unhealthyShoots map[string]bool, timeout time.Duration) error {
	return retry.UntilTimeout(ctx, 1*time.Minute, timeout, func(ctx context.Context) (bool, error) {
		shoots := &gardencorev1beta1.ShootList{}
		err := k8sClient.Client().List(ctx, shoots)
		if err != nil {
			log.V(3).Info(err.Error())
			return retry.MinorError(err)
		}

		reconciledShoots := 0
		for _, shoot := range shoots.Items {

			// ignore shoots that have a do not reconcile label
			if shoot.Annotations[common.ShootIgnore] == "true" {
				reconciledShoots++
				continue
			}

			// ignore shoots that are being deleted
			if shoot.DeletionTimestamp != nil {
				reconciledShoots++
				continue
			}

			if _, ok := unhealthyShoots[telcommon.GetShootKeyFromShoot(&shoot)]; ok {
				reconciledShoots++
				continue
			}

			// check if shoots are in failed state
			// this may need to be adjusted in the future
			if shoot.Status.LastOperation == nil {
				logger.Logf(log.V(3).Info, "shoot %s in namespace %s has no last operation", shoot.Name, shoot.Namespace)
				continue
			}
			if shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateError || shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateFailed || shoot.Status.LastOperation.State == gardencorev1beta1.LastOperationStateAborted {
				reconciledShoots++
				continue
			}

			// check if shoot is not in its maintenance window
			if common.IsNowInEffectiveShootMaintenanceTimeWindow(&shoot) {
				log.V(3).Info("shoot is not in reconcile window", "shoot", shoot.Name, "namespace", shoot.Namespace)
				// check if the last acted gardener version is the current version,
				// to determine if the updated gardener version reconciled the shoot.
				if shoot.Status.Gardener.Version != newGardenerVersion {
					logger.Logf(log.V(3).Info, "last acted gardener version %s does not match current gardener version %s", shoot.Status.Gardener.Version, newGardenerVersion)
					continue
				}
			}

			if framework.ShootCreationCompleted(&shoot) {
				reconciledShoots++
				continue
			}
			logger.Logf(log.V(3).Info, "shoot %s in namespace %s is not completed", shoot.Name, shoot.Namespace)
		}

		if reconciledShoots != len(shoots.Items) {
			err := fmt.Errorf("reconciled %d of %d shoots", reconciledShoots, len(shoots.Items))
			log.Info(err.Error())
			return retry.MinorError(err)
		}

		return retry.Ok()
	})
}
