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

package hostscheduler

import (
	"fmt"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

func ShootKubeconfigSecretName(shootName string) string {
	return fmt.Sprintf("%s.kubeconfig", shootName)
}

func HostKubeconfigPath() (string, error) {
	if tmKubeconfigPath := os.Getenv("TM_KUBECONFIG_PATH"); len(tmKubeconfigPath) != 0 {
		return filepath.Join(os.Getenv("TM_KUBECONFIG_PATH"), "host.config"), nil
	}
	return "", errors.New("TM_KUBECONFIG_PATH is not defined")
}

func HostConfigFilePath() (string, error) {
	if tmKubeconfigPath := os.Getenv("TM_SHARED_PATH"); len(tmKubeconfigPath) != 0 {
		return filepath.Join(os.Getenv("TM_SHARED_PATH"), "host", "config.json"), nil
	}
	return "", errors.New("TM_SHARED_PATH is not defined")
}
