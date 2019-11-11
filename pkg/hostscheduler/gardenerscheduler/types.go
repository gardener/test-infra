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
	"fmt"
	"github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/go-logr/logr"

	"github.com/gardener/gardener/pkg/client/kubernetes"
)

const (
	ShootLabel = "testmachinery.sapcloud.io/host"

	ShootLabelStatus  = "testmachinery.sapcloud.io/status"
	ShootStatusLocked = "locked"
	ShootStatusFree   = "free"

	ShootAnnotationLockedAt = "testmachinery.sapcloud.io/lockedAt"
	ShootAnnotationID       = "testmachinery.sapcloud.io/id"
)

var (
	hibernationTrue  = true
	hibernationFalse = false
)

func ShootKubeconfigSecretName(shootName string) string {
	return fmt.Sprintf("%s.kubeconfig", shootName)
}

type registration struct {
	kubeconfigPath string
	cloudprovider  common.CloudProvider
	scheduler      *gardenerscheduler
}

type gardenerscheduler struct {
	client kubernetes.Interface
	log    logr.Logger

	shootName     string
	namespace     string
	cloudprovider common.CloudProvider
}

func isFree(shoot *v1alpha1.Shoot) bool {
	val, ok := shoot.Labels[ShootLabelStatus]
	if !ok {
		return false
	}

	return val == ShootStatusFree
}

func isLocked(shoot *v1alpha1.Shoot) bool {
	val, ok := shoot.Labels[ShootLabelStatus]
	if !ok {
		return false
	}

	return val == ShootStatusLocked
}
