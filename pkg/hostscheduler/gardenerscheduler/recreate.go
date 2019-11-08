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

package gardenerscheduler

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/types"

	"github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/gardener/pkg/operation/common"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *gardenerscheduler) Recreate(flagset *flag.FlagSet) (hostscheduler.SchedulerFunc, error) {
	name := flagset.String("name", "", "shoot name to recreate")
	force := flagset.BoolP("force", "f", false, "Ignore that a shoot is currently locked. !! Use with care.")
	return func(ctx context.Context) error {
		if name == nil || *name == "" {
			return errors.New("no shoot cluster is defined. Use --name or create a config file")
		}
		shoot := &v1alpha1.Shoot{}
		if err := s.client.Client().Get(ctx, client.ObjectKey{Name: *name, Namespace: s.namespace}, shoot); err != nil {
			return errors.Wrapf(err, "cannot get shoot %s", *name)
		}

		if *force == false {
			if isLocked(shoot) {
				return fmt.Errorf("shoot %s is still in use", *name)
			}
		}

		newShoot := shoot.DeepCopy()
		newShoot.ObjectMeta = metav1.ObjectMeta{
			Name:        shoot.GetName(),
			Namespace:   shoot.GetNamespace(),
			Labels:      shoot.GetLabels(),
			Annotations: shoot.GetAnnotations(),
		}
		newShoot.Status = v1alpha1.ShootStatus{}

		if err := patchAnnotation(ctx, s.client.Client(), shoot, common.ConfirmationDeletion, "true"); err != nil {
			return errors.Wrap(err, "unable to patch deletion confirmation")
		}
		s.log.Info("delete shoot")
		if err := s.client.Client().Delete(ctx, shoot); err != nil {
			return errors.Wrap(err, "unable to delete shoot")
		}
		if err := WaitUntilShootIsDeleted(ctx, s.log, s.client, shoot); err != nil {
			return errors.Wrap(err, "error waiting for shoot to be deleted")
		}
		s.log.Info("shoot successfully deleted")

		// recreate the shoot
		s.log.Info("recreate shoot")
		if err := s.client.Client().Create(ctx, newShoot); err != nil {
			return errors.Wrap(err, "unable to create shoot")
		}
		if _, err := WaitUntilShootIsReconciled(ctx, s.log, s.client, shoot); err != nil {
			return errors.Wrap(err, "error waiting for shoot to be deleted")
		}
		s.log.Info("successfully recreated shoot")

		return nil
	}, nil
}

func patchAnnotation(ctx context.Context, k8sClient client.Client, oldShoot *v1alpha1.Shoot, key, value string) error {
	newShoot := oldShoot.DeepCopy()
	metav1.SetMetaDataAnnotation(&newShoot.ObjectMeta, key, value)
	patchBytes, err := kutil.CreateTwoWayMergePatch(oldShoot, newShoot)
	if err != nil {
		return fmt.Errorf("failed to patch bytes")
	}
	return k8sClient.Patch(ctx, oldShoot, client.ConstantPatch(types.MergePatchType, patchBytes))
}
