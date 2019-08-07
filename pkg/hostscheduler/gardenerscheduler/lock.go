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
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *gardenerscheduler) Lock(ctx context.Context) error {

	shoot := &v1beta1.Shoot{}
	interval := 90 * time.Second
	timeout := 120 * time.Minute

	// try to get an available until the timeout is reached
	return retry.UntilTimeout(ctx, interval, timeout, func(ctx context.Context) (bool, error) {
		var err error
		shoot, err = s.getAvailableHost(ctx, s.logger, s.client, s.namespace)
		if err != nil {
			s.logger.Info("No host available. Trying again...")
			s.logger.Debug(err.Error())
			return retry.MinorError(err)
		}

		s.logger.Infof("shoot %s selected", shoot.Name)

		shoot, err := WaitUntilShootIsReconciled(ctx, s.logger, s.client, shoot)
		if err != nil {
			return retry.SevereError(err)
		}

		s.logger.Infof("Shoot %s was selected as host and will be woken up", shoot.Name)

		if err := downloadHostKubeconfig(ctx, s.logger, s.client, shoot); err != nil {
			s.logger.Error(err.Error())
			return retry.MinorError(err)
		}

		if err := writeHostInformationToFile(s.logger, shoot); err != nil {
			s.logger.Error(err.Error())
			return retry.MinorError(err)
		}
		return true, nil
	})
}

func (s *gardenerscheduler) getAvailableHost(ctx context.Context, logger *logrus.Logger, k8sClient kubernetes.Interface, namespace string) (*v1beta1.Shoot, error) {
	shoots := &v1beta1.ShootList{}
	selector := labels.SelectorFromSet(labels.Set(map[string]string{
		ShootLabel:       "true",
		ShootLabelStatus: ShootStatusFree,
	}))
	err := k8sClient.Client().List(ctx, shoots, client.UseListOptions(&client.ListOptions{
		LabelSelector: selector,
		Namespace:     namespace,
	}))
	if err != nil {
		return nil, fmt.Errorf("shoots cannot be listed: %s", err.Error())
	}

	for _, shoot := range shoots.Items {

		// Try to use the next shoot if the current shoot is not ready.
		if shootReady(&shoot) != nil {
			logger.Debugf("Shoot %s not ready. Skipping...", shoot.Name)
			continue
		}

		if !isHibernated(&shoot) {
			logger.Debugf("Shoot %s not hibernated. Skipping...", shoot.Name)
			continue
		}

		// if shoot is hibernated it is ready to be used as host for a test.
		// then the hibernated shoot is woken up and the gardener tests can start
		shoot.Spec.Hibernation.Enabled = false

		shoot.Labels[ShootLabelStatus] = ShootStatusLocked
		shoot.Annotations[ShootAnnotationLockedAt] = time.Now().String()
		if s.id != "" {
			shoot.Annotations[ShootAnnotationID] = s.id
		}

		err = k8sClient.Client().Update(ctx, &shoot)
		if err != nil {
			// shoot could not be updated, maybe it was concurrently updated by another test.
			// therefore we try the next shoot
			logger.Debugf("Shoot %s cannot be updated. Skipping...", shoot.Name)
			logger.Debug(err.Error())
			continue
		}
		return &shoot, nil
	}
	return nil, fmt.Errorf("cannot find available shoots")
}

func downloadHostKubeconfig(ctx context.Context, logger *logrus.Logger, k8sClient kubernetes.Interface, shoot *v1beta1.Shoot) error {
	// Write kubeconfigPath to kubeconfigPath folder: $TM_KUBECONFIG_PATH/host.config
	logger.Infof("Downloading host kubeconfig to %s", hostscheduler.HostKubeconfigPath())

	// Download kubeconfigPath secret from gardener
	secret := &corev1.Secret{}
	err := k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: ShootKubeconfigSecretName(shoot.Name)}, secret)
	if err != nil {
		return fmt.Errorf("cannot download kubeconfig for shoot %s: %s", shoot.Name, err.Error())
	}

	err = os.MkdirAll(filepath.Dir(hostscheduler.HostKubeconfigPath()), os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot create folder %s for kubeconfig: %s", filepath.Dir(hostscheduler.HostKubeconfigPath()), err.Error())
	}
	err = ioutil.WriteFile(hostscheduler.HostKubeconfigPath(), secret.Data["kubeconfig"], os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot write kubeconfig to %s: %s", hostscheduler.HostKubeconfigPath(), err.Error())
	}

	return nil
}

func writeHostInformationToFile(logger *logrus.Logger, shoot *v1beta1.Shoot) error {
	hostConfig := client.ObjectKey{
		Name:      shoot.Name,
		Namespace: shoot.Namespace,
	}
	data, err := json.Marshal(hostConfig)
	if err != nil {
		logger.Fatalf("cannot unmarshal hostconfig: %s", err.Error())
	}

	err = os.MkdirAll(filepath.Dir(hostscheduler.HostConfigFilePath()), os.ModePerm)
	if err != nil {
		logger.Fatalf("cannot create folder %s for host config: %s", filepath.Dir(hostscheduler.HostConfigFilePath()), err.Error())
	}
	err = ioutil.WriteFile(hostscheduler.HostConfigFilePath(), data, os.ModePerm)
	if err != nil {
		logger.Fatalf("cannot write host config to %s: %s", hostscheduler.HostConfigFilePath(), err.Error())
	}

	return nil
}
