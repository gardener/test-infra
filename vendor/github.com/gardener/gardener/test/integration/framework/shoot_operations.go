// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package framework

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/gardener/pkg/utils/retry"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sirupsen/logrus"

	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/operation/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewShootGardenerTest creates a new shootGardenerTest object, given an already created shoot (created after parsing a shoot YAML)
func NewShootGardenerTest(kubeconfig string, shoot *v1beta1.Shoot, logger *logrus.Logger) (*ShootGardenerTest, error) {
	if len(kubeconfig) == 0 {
		return nil, fmt.Errorf("please specify the kubeconfig path correctly")
	}

	k8sGardenClient, err := kubernetes.NewClientFromFile("", kubeconfig, kubernetes.WithClientOptions(
		client.Options{
			Scheme: kubernetes.GardenScheme,
		}),
	)
	if err != nil {
		return nil, err
	}

	return &ShootGardenerTest{
		GardenClient: k8sGardenClient,

		Shoot:  shoot,
		Logger: logger,
	}, nil
}

// GetShoot gets the test shoot
func (s *ShootGardenerTest) GetShoot(ctx context.Context) (*v1beta1.Shoot, error) {
	shoot := &v1beta1.Shoot{}
	err := s.GardenClient.Client().Get(ctx, client.ObjectKey{
		Namespace: s.Shoot.Namespace,
		Name:      s.Shoot.Name,
	}, shoot)

	if err != nil {
		return nil, err
	}
	return shoot, err
}

// CreateShootResource creates a shoot from a shoot Object
func (s *ShootGardenerTest) CreateShootResource(ctx context.Context, shootToCreate *v1beta1.Shoot) (*v1beta1.Shoot, error) {
	shoot := s.Shoot
	if err := s.GardenClient.Client().Create(ctx, shoot); err != nil {
		return nil, err
	}

	s.Logger.Infof("Shoot resource %s was created!", shoot.Name)
	return shoot, nil
}

// CreateShoot Creates a shoot from a shoot Object and waits until it is successfully reconciled
func (s *ShootGardenerTest) CreateShoot(ctx context.Context) (*v1beta1.Shoot, error) {
	_, err := s.GetShoot(ctx)
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	shoot := s.Shoot
	err = retry.UntilTimeout(ctx, 20*time.Second, 5*time.Minute, func(ctx context.Context) (done bool, err error) {
		_, err = s.CreateShootResource(ctx, shoot)
		if err != nil {
			s.Logger.Debugf("unable to create shoot %s: %s", shoot.Name, err.Error())
			return retry.MinorError(err)
		}
		return retry.Ok()
	})
	if err != nil {
		return nil, err
	}

	// Then we wait for the shoot to be created
	err = s.WaitForShootToBeCreated(ctx)
	if err != nil {
		return nil, err
	}

	s.Logger.Infof("Shoot %s was created!", shoot.Name)
	return shoot, nil
}

// UpdateShoot Updates a shoot from a shoot Object and waits for its reconciliation
func (s *ShootGardenerTest) UpdateShoot(ctx context.Context, newShoot *v1beta1.Shoot) (*v1beta1.Shoot, error) {
	err := retry.UntilTimeout(ctx, 20*time.Second, 5*time.Minute, func(ctx context.Context) (done bool, err error) {
		err = s.GardenClient.Client().Update(ctx, newShoot)
		if err != nil {
			s.Logger.Debugf("unable to update shoot %s: %s", newShoot.Name, err.Error())
			return retry.MinorError(err)
		}
		return retry.Ok()
	})
	if err != nil {
		return nil, err
	}

	s.Shoot = newShoot

	// Then we wait for the shoot to be created
	err = s.WaitForShootToBeReconciled(ctx)
	if err != nil {
		return nil, err
	}

	s.Logger.Infof("Shoot %s was successfully updated!", newShoot.Name)
	return s.Shoot, nil
}

// DeleteShoot deletes the test shoot
func (s *ShootGardenerTest) DeleteShoot(ctx context.Context) error {
	shoot, err := s.GetShoot(ctx)
	if err != nil {
		return err
	}

	s.Shoot = shoot
	err = retry.UntilTimeout(ctx, 20*time.Second, 5*time.Minute, func(ctx context.Context) (done bool, err error) {
		err = s.RemoveShootAnnotation(ctx, common.ShootIgnore)
		if err != nil {
			return retry.MinorError(err)
		}

		// First we annotate the shoot to be deleted.
		err = s.AnnotateShoot(ctx, map[string]string{
			common.ConfirmationDeletion: "true",
		})
		if err != nil {
			return retry.MinorError(err)
		}

		err = s.GardenClient.Client().Delete(ctx, s.Shoot)
		if err != nil {
			return retry.MinorError(err)
		}

		return retry.Ok()
	})
	if err != nil {
		return err
	}

	err = s.WaitForShootToBeDeleted(ctx)
	if err != nil {
		return err
	}

	s.Logger.Infof("Shoot %s was deleted successfully!", s.Shoot.Name)
	return nil
}

// HibernateShoot hibernates the test shoot
func (s *ShootGardenerTest) HibernateShoot(ctx context.Context) error {
	shoot, err := s.GetShoot(ctx)
	if err != nil {
		return err
	}
	s.Shoot = shoot

	// return if the shoot is already hibernated
	if s.Shoot.Spec.Hibernation != nil && *s.Shoot.Spec.Hibernation.Enabled {
		return nil
	}

	err = retry.UntilTimeout(ctx, 20*time.Second, 5*time.Minute, func(ctx context.Context) (done bool, err error) {
		setHibernation(s.Shoot, true)

		err = s.GardenClient.Client().Update(ctx, s.Shoot)
		if err != nil {
			return retry.MinorError(err)
		}

		return retry.Ok()
	})
	if err != nil {
		return err
	}

	err = s.WaitForShootToBeReconciled(ctx)
	if err != nil {
		return err
	}

	s.Logger.Infof("Shoot %s was hibernated successfully!", s.Shoot.Name)
	return nil
}

// WakeUpShoot wakes up the test shoot from hibernation
func (s *ShootGardenerTest) WakeUpShoot(ctx context.Context) error {
	shoot, err := s.GetShoot(ctx)
	if err != nil {
		return err
	}
	s.Shoot = shoot

	// return if the shoot is already running
	if s.Shoot.Spec.Hibernation == nil || !*s.Shoot.Spec.Hibernation.Enabled {
		return nil
	}

	err = retry.UntilTimeout(ctx, 20*time.Second, 5*time.Minute, func(ctx context.Context) (done bool, err error) {
		setHibernation(s.Shoot, false)

		err = s.GardenClient.Client().Update(ctx, s.Shoot)
		if err != nil {
			return retry.MinorError(err)
		}

		return retry.Ok()
	})
	if err != nil {
		return err
	}

	err = s.WaitForShootToBeReconciled(ctx)
	if err != nil {
		return err
	}

	s.Logger.Infof("Shoot %s has been woken up successfully!", s.Shoot.Name)
	return nil
}

// RemoveShootAnnotation removes an annotation with key <annotationKey> from a shoot object
func (s *ShootGardenerTest) RemoveShootAnnotation(ctx context.Context, annotationKey string) error {
	shootCopy := s.Shoot.DeepCopy()
	if len(shootCopy.Annotations) == 0 {
		return nil
	}
	if _, ok := shootCopy.Annotations[annotationKey]; !ok {
		return nil
	}

	// start the update process with Kubernetes
	s.Logger.Infof("deleting annotation with key: %q in shoot: %s\n", annotationKey, s.Shoot.Name)
	delete(shootCopy.Annotations, annotationKey)

	return s.mergePatch(ctx, s.Shoot, shootCopy)
}

// AnnotateShoot adds shoot annotation(s)
func (s *ShootGardenerTest) AnnotateShoot(ctx context.Context, annotations map[string]string) error {
	shootCopy := s.Shoot.DeepCopy()

	for annotationKey, annotationValue := range annotations {
		metav1.SetMetaDataAnnotation(&shootCopy.ObjectMeta, annotationKey, annotationValue)
	}

	err := s.mergePatch(ctx, s.Shoot, shootCopy)
	if err != nil {
		return err
	}

	return nil
}

// WaitForShootToBeCreated waits for the shoot to be created
func (s *ShootGardenerTest) WaitForShootToBeCreated(ctx context.Context) error {
	return retry.Until(ctx, 30*time.Second, func(ctx context.Context) (done bool, err error) {
		shoot := &v1beta1.Shoot{}
		err = s.GardenClient.Client().Get(ctx, client.ObjectKey{Namespace: s.Shoot.Namespace, Name: s.Shoot.Name}, shoot)
		if err != nil {
			s.Logger.Infof("Error while waiting for shoot to be created: %s", err.Error())
			return retry.MinorError(err)
		}
		if ShootCreationCompleted(shoot) {
			return retry.Ok()
		}
		s.Logger.Infof("Waiting for shoot %s to be created", s.Shoot.Name)
		if shoot.Status.LastOperation != nil {
			s.Logger.Debugf("%d%%: Shoot State: %s, Description: %s", shoot.Status.LastOperation.Progress, shoot.Status.LastOperation.State, shoot.Status.LastOperation.Description)
		}
		return retry.MinorError(fmt.Errorf("shoot %q was not successfully reconciled", shoot.Name))
	})
}

// WaitForShootToBeReconciled waits for the shoot to be successfully reconciled
func (s *ShootGardenerTest) WaitForShootToBeReconciled(ctx context.Context) error {
	return retry.Until(ctx, 30*time.Second, func(ctx context.Context) (done bool, err error) {
		shoot := &v1beta1.Shoot{}
		err = s.GardenClient.Client().Get(ctx, client.ObjectKey{Namespace: s.Shoot.Namespace, Name: s.Shoot.Name}, shoot)
		if err != nil {
			s.Logger.Infof("Error while waiting for shoot to be reconciled: %s", err.Error())
			return retry.MinorError(err)
		}
		if ShootCreationCompleted(shoot) {
			return retry.Ok()
		}
		s.Logger.Infof("Waiting for shoot %s to be reconciled", s.Shoot.Name)
		if shoot.Status.LastOperation != nil {
			s.Logger.Debugf("%d%%: Shoot State: %s, Description: %s", shoot.Status.LastOperation.Progress, shoot.Status.LastOperation.State, shoot.Status.LastOperation.Description)
		}
		return retry.MinorError(fmt.Errorf("shoot %q was not successfully reconciled", shoot.Name))
	})
}

// WaitForShootToBeDeleted waits for the shoot to be deleted
func (s *ShootGardenerTest) WaitForShootToBeDeleted(ctx context.Context) error {
	return retry.Until(ctx, 30*time.Second, func(ctx context.Context) (done bool, err error) {
		shoot := &v1beta1.Shoot{}
		err = s.GardenClient.Client().Get(ctx, client.ObjectKey{Namespace: s.Shoot.Namespace, Name: s.Shoot.Name}, shoot)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return retry.Ok()
			}
			s.Logger.Infof("Error while waiting for shoot to be deleted: %s", err.Error())
			return retry.MinorError(err)
		}
		s.Logger.Infof("waiting for shoot %s to be deleted", s.Shoot.Name)
		if shoot.Status.LastOperation != nil {
			s.Logger.Debugf("%d%%: Shoot state: %s, Description: %s", shoot.Status.LastOperation.Progress, shoot.Status.LastOperation.State, shoot.Status.LastOperation.Description)
		}
		return retry.MinorError(fmt.Errorf("shoot %q still exists", shoot.Name))
	})
}
