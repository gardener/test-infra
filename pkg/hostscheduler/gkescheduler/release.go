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
	"github.com/pkg/errors"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

func (s gkescheduler) Release(ctx context.Context) error {

	hostName, err := readHostInformationFromFile()
	if err != nil {
		return err
	}

	cluster, err := s.client.GetCluster(ctx, &containerpb.GetClusterRequest{Name: s.getClusterName(hostName)})
	if err != nil {
		return err
	}
	s.logger.Infof("starting to release cluster %s", cluster.GetName())

	// scale all non-default node pools to 0
	for _, nodepool := range cluster.NodePools {
		if nodepool.GetName() != GKEDefaultNodePoolName {
			s.logger.Infof("scale down node pool %s to 0", nodepool.GetName())
			o, err := s.client.SetNodePoolSize(ctx, &containerpb.SetNodePoolSizeRequest{
				NodeCount: 0,
				Name:      s.getNodePoolName(cluster.GetName(), nodepool.GetName()),
			})
			if err != nil {
				return errors.Wrapf(err, "unable to scale node pool %s of cluster %s down to 0", nodepool.GetName(), cluster.GetName())
			}
			if err := s.waitUntilOperationFinishedSuccessfully(ctx, o); err != nil {
				return errors.Wrapf(err, "waiting for operation %s to finish. unable to scale node pool %s of cluster %s down to 0.", o.GetName(), nodepool.GetName(), cluster.GetName())
			}
		}
	}
	s.logger.Info("all non-default nodepools scaled down successfully")

	labels := cluster.GetResourceLabels()

	// directly return if the shoot is already freed
	if isFree(labels) {
		return nil
	}

	s.logger.Info("update labels of cluster")
	labels[ClusterStatusLabel] = ClusterStatusFree
	delete(labels, ClusterLockedAtLabel)

	labelsRequest := &containerpb.SetLabelsRequest{
		Name:           s.getClusterName(hostName),
		ResourceLabels: labels,
	}
	o, err := s.client.SetLabels(ctx, labelsRequest)
	if err != nil {
		return errors.Wrapf(err, "unable to update labels for cluster %s", cluster.GetName())
	}

	return s.waitUntilOperationFinishedSuccessfully(ctx, o)
}
