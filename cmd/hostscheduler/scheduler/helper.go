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

package scheduler

import (
	"context"
	"fmt"
	"time"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitUntilShootIsReconciled waits until a cluster is reconciled and ready to use
func WaitUntilShootIsReconciled(ctx context.Context, k8sClient kubernetes.Interface, shoot *v1beta1.Shoot) (*v1beta1.Shoot, error) {
	interval := 1 * time.Minute
	timeout := 30 * time.Minute
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		shootObject := &v1beta1.Shoot{}
		err := k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: shoot.Name}, shootObject)
		if err != nil {
			log.Infof("Wait for shoot to be reconciled...")
			log.Debug(err.Error())
			return false, nil
		}
		shoot = shootObject
		if err := shootReady(shoot); err != nil {
			log.Infof("%s. Wait for shoot to be reconciled...", err.Error())
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return shoot, nil
}

func shootReady(newShoot *v1beta1.Shoot) error {
	newStatus := newShoot.Status
	if len(newStatus.Conditions) == 0 {
		return fmt.Errorf("no conditions in newShoot status")
	}

	if newShoot.Generation != newStatus.ObservedGeneration {
		return fmt.Errorf("observed generation is unlike newShoot generation")
	}

	for _, condition := range newStatus.Conditions {
		if condition.Status != gardencorev1alpha1.ConditionTrue {
			return fmt.Errorf("condition of %s is %s", condition.Type, condition.Status)
		}
	}

	if newStatus.LastOperation != nil {
		if newStatus.LastOperation.Type == gardencorev1alpha1.LastOperationTypeCreate ||
			newStatus.LastOperation.Type == gardencorev1alpha1.LastOperationTypeReconcile {
			if newStatus.LastOperation.State != gardencorev1alpha1.LastOperationStateSucceeded {
				return fmt.Errorf("last operation %s is %s", newStatus.LastOperation.Type, newStatus.LastOperation.State)
			}
		}
	}

	return nil
}
