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
	"fmt"

	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

func (s *gkescheduler) Release(flagset *flag.FlagSet) (hostscheduler.SchedulerFunc, error) {

	return func(ctx context.Context) error {
		if len(s.hostname) == 0 {
			var err error
			s.hostname, err = readHostInformationFromFile()
			if err != nil {
				s.log.V(3).Info(err.Error())
				return errors.New("no shoot cluster is defined. Use --name or create a config file")
			}
		}
		s.log = s.log.WithValues("host", s.getClusterName(s.hostname))

		cluster, err := s.client.GetCluster(ctx, &containerpb.GetClusterRequest{Name: s.getClusterName(s.hostname)})
		if err != nil {
			return err
		}
		s.log.Info("starting to release cluster")

		// scale all non-default node pools to 0
		for _, nodepool := range cluster.NodePools {
			if nodepool.GetName() != GKEDefaultNodePoolName {
				s.log.Info(fmt.Sprintf("scale down node pool %s to 0", nodepool.GetName()))
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
		s.log.Info("all non-default nodepools scaled down successfully")

		labels := cluster.GetResourceLabels()

		// directly return if the shoot is already freed
		if isFree(labels) {
			return nil
		}

		s.log.Info("update labels of cluster")
		labels[ClusterStatusLabel] = ClusterStatusFree
		delete(labels, ClusterLockedAtLabel)
		delete(labels, ClusterLabelID)

		labelsRequest := &containerpb.SetLabelsRequest{
			Name:             s.getClusterName(s.hostname),
			ResourceLabels:   labels,
			LabelFingerprint: cluster.LabelFingerprint,
		}
		o, err := s.client.SetLabels(ctx, labelsRequest)
		if err != nil {
			return errors.Wrapf(err, "unable to update labels for cluster %s", cluster.GetName())
		}

		return s.waitUntilOperationFinishedSuccessfully(ctx, o)
	}, nil
}
