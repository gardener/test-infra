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
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"

	flag "github.com/spf13/pflag"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *gardenerscheduler) Lock(flagset *flag.FlagSet) (hostscheduler.SchedulerFunc, error) {
	id := flagset.String("id", "", "Unique id to identify the cluster")
	return func(ctx context.Context) error {
		shoot := &gardencorev1beta1.Shoot{}
		interval := 90 * time.Second
		timeout := 120 * time.Minute

		// try to get an available host until the timeout is reached
		return retry.UntilTimeout(ctx, interval, timeout, func(ctx context.Context) (bool, error) {
			var err error

			if s.shootName == "" {
				shoot, err = s.getAvailableHost(ctx)
				if err != nil {
					s.log.Info("No host available. Trying again...")
					s.log.V(3).Info(err.Error())
					return retry.MinorError(err)
				}
			} else {
				err := s.client.Client().Get(ctx, client.ObjectKey{Name: s.shootName, Namespace: s.namespace}, shoot)
				if err != nil {
					s.log.V(3).Info(err.Error())
					if apierrors.IsNotFound(err) {
						return retry.SevereError(err)
					}
					s.log.Info("No host available. Trying again...")
					return retry.MinorError(err)
				}
			}

			if err := s.lockShoot(ctx, shoot, *id); err != nil {
				// shoot could not be updated, maybe it was concurrently updated by another test.
				// therefore we try the next shoot
				s.log.V(3).Info("Shoot cannot be updated. Skipping...", "shoot", shoot.Name)
				s.log.V(3).Info(err.Error())
				return retry.MinorError(err)
			}

			s.log.Info(fmt.Sprintf("shoot %s selected", shoot.Name))
			s.log = s.log.WithValues("shoot", shoot.Name, "namespace", shoot.Namespace)

			shoot, err := WaitUntilShootIsReconciled(ctx, s.log, s.client, shoot)
			if err != nil {
				return retry.SevereError(err)
			}

			s.log.Info("Shoot was selected as host and woken up")

			if err := downloadHostKubeconfig(ctx, s.log, s.client, shoot); err != nil {
				s.log.Error(err, "unable to download kubeconfig")
				return retry.MinorError(err)
			}

			if err := writeHostInformationToFile(s.log, shoot); err != nil {
				s.log.Error(err, "unable to write host information to file")
				return retry.MinorError(err)
			}
			return true, nil
		})
	}, nil
}

func (s *gardenerscheduler) getAvailableHost(ctx context.Context) (*gardencorev1beta1.Shoot, error) {
	shoots := &gardencorev1beta1.ShootList{}
	selector := labels.SelectorFromSet(labels.Set(map[string]string{
		ShootLabel:       "true",
		ShootLabelStatus: ShootStatusFree,
	}))
	err := s.client.Client().List(ctx, shoots, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     s.namespace,
	})
	if err != nil {
		return nil, fmt.Errorf("shoots cannot be listed: %s", err.Error())
	}

	for _, shoot := range shoots.Items {

		// check if the cloudprovider matches
		if s.cloudprovider != CloudProviderAll {
			if shoot.Spec.Provider.Type != string(s.cloudprovider) {
				s.log.V(3).Info(fmt.Sprintf("found shoot from cloudprovider %s but want cloudprovider %s", shoot.Spec.Provider.Type, s.cloudprovider), "shoot", shoot.Name)
				continue
			}
		}

		// Try to use the next shoot if the current shoot is not ready.
		if shootReady(&shoot) != nil {
			s.log.V(3).Info("Shoot is not ready. Skipping...", "shoot", shoot.Name)
			continue
		}

		if !isHibernated(&shoot) {
			s.log.V(3).Info("Shoot not hibernated. Skipping...", "shoot", shoot.Name)
			continue
		}

		return &shoot, nil
	}
	return nil, fmt.Errorf("cannot find available shoots")
}

func (s *gardenerscheduler) lockShoot(ctx context.Context, shoot *gardencorev1beta1.Shoot, id string) error {
	// if shoot is hibernated it is ready to be used as host for a test.
	// then the hibernated shoot is woken up and the gardener tests can start
	shoot.Spec.Hibernation.Enabled = &hibernationFalse

	shoot.Labels[ShootLabelStatus] = ShootStatusLocked
	shoot.Annotations[ShootAnnotationLockedAt] = time.Now().Format(time.RFC3339)
	if id != "" {
		shoot.Annotations[ShootAnnotationID] = id
	}

	err := s.client.Client().Update(ctx, shoot)
	if err != nil {
		return errors.Wrapf(err, "shoot cannot be updated")
	}
	return nil
}

func downloadHostKubeconfig(ctx context.Context, logger logr.Logger, k8sClient kubernetes.Interface, shoot *gardencorev1beta1.Shoot) error {
	// Write kubeconfigPath to kubeconfigPath folder: $TM_KUBECONFIG_PATH/host.config
	kubeconfigPath, err := hostscheduler.HostKubeconfigPath()
	if err != nil {
		logger.V(3).Info(fmt.Sprintf("kubeconfig is not downloaded: %s", err.Error()))
		return nil
	}
	logger.Info(fmt.Sprintf("Downloading host kubeconfig to %s", kubeconfigPath))

	// Download kubeconfigPath secret from gardener
	secret := &corev1.Secret{}
	err = k8sClient.Client().Get(ctx, client.ObjectKey{Namespace: shoot.Namespace, Name: ShootKubeconfigSecretName(shoot.Name)}, secret)
	if err != nil {
		return fmt.Errorf("cannot download kubeconfig for shoot %s: %s", shoot.Name, err.Error())
	}

	err = os.MkdirAll(filepath.Dir(kubeconfigPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot create folder %s for kubeconfig: %s", filepath.Dir(kubeconfigPath), err.Error())
	}
	err = ioutil.WriteFile(kubeconfigPath, secret.Data["kubeconfig"], os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot write kubeconfig to %s: %s", kubeconfigPath, err.Error())
	}

	return nil
}

func writeHostInformationToFile(log logr.Logger, shoot *gardencorev1beta1.Shoot) error {
	hostConfigPath, err := hostscheduler.HostConfigFilePath()
	if err != nil {
		log.V(3).Info("hostconfig is not written", "error", err.Error())
		return nil
	}

	hostConfig := client.ObjectKey{
		Name:      shoot.Name,
		Namespace: shoot.Namespace,
	}
	data, err := json.Marshal(hostConfig)
	if err != nil {
		return fmt.Errorf("cannot unmarshal hostconfig: %s", err.Error())
	}

	err = os.MkdirAll(filepath.Dir(hostConfigPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot create folder %s for host config: %s", filepath.Dir(hostConfigPath), err.Error())
	}
	err = ioutil.WriteFile(hostConfigPath, data, os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot write host config to %s: %s", hostConfigPath, err.Error())
	}

	return nil
}
