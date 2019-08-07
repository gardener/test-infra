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
	"encoding/json"
	"fmt"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"io/ioutil"
	"time"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func isHibernated(shoot *v1beta1.Shoot) bool {
	if shoot.Spec.Hibernation == nil {
		return false
	}
	return shoot.Spec.Hibernation.Enabled
}

func getNamespaceOfKubeconfig(kubeconfigPath string) (string, error) {
	data, err := ioutil.ReadFile(kubeconfigPath)
	if err != nil {
		return "", errors.Wrapf(err, "cannot read file from %s", kubeconfigPath)
	}
	cfg, err := clientcmd.NewClientConfigFromBytes(data)
	if err != nil {
		return "", err
	}

	ns, _, err := cfg.Namespace()
	if err != nil {
		return "", err
	}
	return ns, nil
}

// WaitUntilShootIsReconciled waits until a cluster is reconciled and ready to use
func WaitUntilShootIsReconciled(ctx context.Context, logger *logrus.Logger, k8sClient kubernetes.Interface, shoot *v1beta1.Shoot) (*v1beta1.Shoot, error) {
	interval := 1 * time.Minute
	timeout := 30 * time.Minute
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		shootObject := &v1beta1.Shoot{}
		err := k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: shoot.Name}, shootObject)
		if err != nil {
			logger.Infof("Wait for shoot to be reconciled...")
			logger.Debug(err.Error())
			return false, nil
		}
		shoot = shootObject
		if err := shootReady(shoot); err != nil {
			logger.Infof("%s. Wait for shoot to be reconciled...", err.Error())
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

func readHostInformationFromFile() (*client.ObjectKey, error) {
	data, err := ioutil.ReadFile(hostscheduler.HostConfigFilePath())
	if err != nil {
		return nil, fmt.Errorf("cannot read file %s: %s", hostscheduler.HostConfigFilePath(), err.Error())
	}

	var hostConfig client.ObjectKey
	err = json.Unmarshal(data, &hostConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal host config: %s", err.Error())
	}

	return &hostConfig, nil
}
