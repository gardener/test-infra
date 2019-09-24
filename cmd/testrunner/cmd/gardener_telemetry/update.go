package gardener_telemetry_cmd

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/test-infra/pkg/logger"
	telcommon "github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"time"

	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/gardener/gardener/test/integration/framework"
	"github.com/go-logr/logr"
)

func GetUnhealthyShoots(log logr.Logger, ctx context.Context, k8sClient kubernetes.Interface) (map[string]bool, error) {
	shoots := &v1beta1.ShootList{}
	err := k8sClient.Client().List(ctx, shoots)
	if err != nil {
		return nil, err
	}

	unhealthyShoots := make(map[string]bool)
	for _, shoot := range shoots.Items {
		if shoot.Status.LastOperation.State == v1alpha1.LastOperationStateError || shoot.Status.LastOperation.State == v1alpha1.LastOperationStateFailed || shoot.Status.LastOperation.State == v1alpha1.LastOperationStateAborted {
			unhealthyShoots[telcommon.GetShootKeyFromShoot(&shoot)] = true
			continue
		}
		if shoot.Status.LastError != nil {
			unhealthyShoots[telcommon.GetShootKeyFromShoot(&shoot)] = true
		}
	}

	return unhealthyShoots, nil
}

func WaitForGardenerUpdate(log logr.Logger, ctx context.Context, k8sClient kubernetes.Interface, newGardenerVersion string, unhealthyShoots map[string]bool, timeout time.Duration) error {
	return retry.UntilTimeout(ctx, 30*time.Second, timeout, func(ctx context.Context) (bool, error) {
		shoots := &v1beta1.ShootList{}
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
			if shoot.Status.LastOperation.State == v1alpha1.LastOperationStateError || shoot.Status.LastOperation.State == v1alpha1.LastOperationStateFailed || shoot.Status.LastOperation.State == v1alpha1.LastOperationStateAborted {
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
