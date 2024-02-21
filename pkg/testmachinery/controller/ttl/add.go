// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package ttl

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/gardener/test-infra/pkg/apis/config"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// AddControllerToManager adds a new ttl controller to the manager.
func AddControllerToManager(log logr.Logger, mgr manager.Manager, config config.TTLController) error {
	c := New(log.WithName("TTL"), mgr.GetClient(), mgr.GetScheme())

	bldr := ctrl.NewControllerManagedBy(mgr).For(&tmv1beta1.Testrun{})
	if config.MaxConcurrentSyncs != 0 {
		bldr.WithOptions(controller.Options{
			MaxConcurrentReconciles: config.MaxConcurrentSyncs,
		})
	}
	return bldr.Complete(c)
}
