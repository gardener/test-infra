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

package gkescheduler

import (
	"cloud.google.com/go/container/apiv1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/sirupsen/logrus"
)

const (
	ClusterLabel = "tm-host"

	ClusterLockedAtLabel = "tm-host-lockedat"

	ClusterStatusLabel  = "tm-host-status"
	ClusterStatusLocked = "locked"
	ClusterStatusFree   = "free"

	GKEDefaultNodePoolName = "default-pool"
)

type hostCluster struct {
	Name   string
	Client kubernetes.Interface
}

type gkescheduler struct {
	client *container.ClusterManagerClient
	logger *logrus.Logger

	project string
	zone    string
}

type config struct {
	Name string `json:"name"`
}

func isHost(labels map[string]string) bool {
	_, ok := labels[ClusterLabel]
	return ok
}

func isFree(labels map[string]string) bool {
	if val, ok := labels[ClusterStatusLabel]; ok {
		if val == ClusterStatusFree {
			return true
		}
	}
	return false
}
func isLocked(labels map[string]string) bool {
	if val, ok := labels[ClusterStatusLabel]; ok {
		if val == ClusterStatusLocked {
			return true
		}
	}
	return false
}
