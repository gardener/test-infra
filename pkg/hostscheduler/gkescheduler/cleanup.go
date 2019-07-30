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
	"context"
	"github.com/gardener/test-infra/pkg/hostscheduler/cleanup"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

func (s *gkescheduler) Cleanup(ctx context.Context) error {
	hostName, err := readHostInformationFromFile()
	if err != nil {
		return err
	}

	cluster, err := s.client.GetCluster(ctx, &containerpb.GetClusterRequest{Name: s.getClusterName(hostName)})
	if err != nil {
		return err
	}

	// do not cleanup if cluster is already freed
	if isFree(cluster.GetResourceLabels()) {
		s.logger.Infof("Cluster %s is already free. No need to cleanup.", hostName)
		return nil
	}

	k8sClient, err := clientFromCluster(cluster)
	if err != nil {
		return err
	}

	if err := cleanup.CleanResources(ctx, s.logger, k8sClient); err != nil {
		return err
	}
	return nil
}
