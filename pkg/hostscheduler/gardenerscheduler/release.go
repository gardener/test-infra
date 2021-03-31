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

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/hostscheduler"
)

func (s *gardenerscheduler) Release(flagset *flag.FlagSet) (hostscheduler.SchedulerFunc, error) {
	return func(ctx context.Context) error {
		var (
			err        error
			hostConfig = &client.ObjectKey{Name: s.shootName, Namespace: s.namespace}
		)
		if len(s.shootName) == 0 {
			hostConfig, err = readHostInformationFromFile()
			if err != nil {
				s.log.V(3).Info(err.Error())
				return errors.New("no shoot cluster is defined. Use --name or create a config file")
			}
		}

		shoot := &gardencorev1beta1.Shoot{}
		if err := s.client.Get(ctx, client.ObjectKey{Namespace: hostConfig.Namespace, Name: hostConfig.Name}, shoot); err != nil {
			return fmt.Errorf("cannot get shoot %s: %s", hostConfig.Name, err.Error())
		}

		shoot, err = WaitUntilShootIsReconciled(ctx, s.log, s.client, shoot)
		if err != nil {
			return fmt.Errorf("cannot hibernate shoot %s: %s", shoot.Name, err.Error())
		}

		if isFree(shoot) && isHibernated(shoot) {
			s.log.V(3).Info("Shoot is already free and hibernated")
			return nil
		}

		// Do not set any hibernation schedule as hibernation should be handled automatically by this hostscheduler.

		newShoot := shoot.DeepCopy()
		newShoot.Spec.Hibernation = &gardencorev1beta1.Hibernation{Enabled: &hibernationTrue}
		newShoot.Labels[ShootLabelStatus] = ShootStatusFree
		delete(newShoot.Annotations, ShootAnnotationLockedAt)
		delete(newShoot.Annotations, ShootAnnotationID)
		if err := s.client.Patch(ctx, newShoot, client.MergeFrom(shoot)); err != nil {
			return fmt.Errorf("cannot hibernate shoot %s: %s", shoot.Name, err.Error())
		}

		shoot, err = WaitUntilShootIsReconciled(ctx, s.log, s.client, shoot)
		if err != nil {
			return fmt.Errorf("cannot hibernate shoot %s: %s", shoot.Name, err.Error())
		}
		return nil
	}, nil
}
