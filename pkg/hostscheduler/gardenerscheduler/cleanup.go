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
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/hostscheduler/cleanup"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *gardenerscheduler) Cleanup(ctx context.Context) error {

	hostConfig, err := readHostInformationFromFile()
	if err != nil {
		return err
	}

	shoot := &v1beta1.Shoot{}
	err = s.client.Client().Get(ctx, client.ObjectKey{Namespace: hostConfig.Namespace, Name: hostConfig.Name}, shoot)
	if err != nil {
		return fmt.Errorf("cannot get shoot %s: %s", hostConfig.Name, err.Error())
	}

	hostClient, err := kubernetes.NewClientFromSecret(s.client, hostConfig.Namespace, ShootKubeconfigSecretName(shoot.Name), client.Options{
		Scheme: kubernetes.ShootScheme,
	})
	if err != nil {
		return fmt.Errorf("cannot build shoot client: %s", err.Error())
	}

	shoot, err = WaitUntilShootIsReconciled(ctx, s.logger, s.client, shoot)
	if err != nil {
		return fmt.Errorf("cannot reconcile shoot %s: %s", shoot.Name, err.Error())
	}

	if shoot.Spec.Hibernation != nil && shoot.Spec.Hibernation.Enabled {
		s.logger.Infof("Cluster %s is already free. No need to cleanup.", shoot.Name)
		return nil
	}

	if err := cleanup.CleanResources(ctx, s.logger, hostClient); err != nil {
		return err
	}

	return nil
}
