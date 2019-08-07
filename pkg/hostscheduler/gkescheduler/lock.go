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
	container "cloud.google.com/go/container/apiv1"
	"context"
	"errors"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/sirupsen/logrus"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"strconv"
	"time"
)

const (
	SelectHostTimeout = 2 * time.Hour
)

func (s gkescheduler) Lock(ctx context.Context) error {

	host, err := s.selectAvailableHost(ctx)
	if err != nil {
		return err
	}

	req := &containerpb.GetClusterRequest{
		Name: s.getClusterName(host.Name),
	}

	cluster, err := s.client.GetCluster(ctx, req)
	if err != nil {
		return err
	}

	labels := cluster.GetResourceLabels()
	labels[ClusterStatusLabel] = ClusterStatusLocked
	labels[ClusterLockedAtLabel] = strconv.FormatInt(time.Now().Unix(), 10)
	if s.id != "" {
		labels[ClusterLabelID] = s.id
	}

	labelsRequest := &containerpb.SetLabelsRequest{
		Name:           s.getClusterName(host.Name),
		ResourceLabels: labels,
	}
	o, err := s.client.SetLabels(ctx, labelsRequest)
	if err != nil {
		return err
	}

	if err := s.waitUntilOperationFinishedSuccessfully(ctx, o); err != nil {
		return err
	}

	if err := hostscheduler.WriteHostKubeconfig(host.Client); err != nil {
		return err
	}

	return writeHostInformationToFile(host)
}

func (s gkescheduler) selectAvailableHost(ctx context.Context) (*hostCluster, error) {
	var (
		host *hostCluster
		err  error
	)
	err = retry.UntilTimeout(ctx, 1*time.Minute, SelectHostTimeout, func(ctx context.Context) (bool, error) {
		host, err = tryAvailableHost(ctx, s.logger, s.client, s.getParentName())
		if err != nil {
			s.logger.Debug(err.Error())
			s.logger.Info("Unable to select host cluster. Trying again..")
			return retry.MinorError(err)
		}
		return retry.Ok()
	})
	return host, err
}

func tryAvailableHost(ctx context.Context, logger *logrus.Logger, client *container.ClusterManagerClient, parent string) (*hostCluster, error) {
	req := &containerpb.ListClustersRequest{
		Parent: parent,
	}
	resp, err := client.ListClusters(ctx, req)
	if err != nil {
		return nil, err
	}

	for _, cluster := range resp.Clusters {
		if cluster.GetStatus() != containerpb.Cluster_RUNNING {
			logger.Debugf("found %s but cluster state is %s", cluster.Name, containerpb.Cluster_Status_name[int32(cluster.GetStatus())])
			continue
		}
		labels := cluster.GetResourceLabels()
		if !isHost(labels) {
			logger.Debugf("found %s but cluster is no host cluster with label %s", cluster.GetName(), ClusterLabel)
			continue
		}
		if isLocked(labels) {
			logger.Debugf("found %s but cluster is locked", cluster.GetName())
			continue
		}

		logger.Infof("Use gke cluster %s", cluster.GetName())
		k8sClient, err := clientFromCluster(cluster)
		if err != nil {
			logger.Debug(err.Error())
			continue
		}

		return &hostCluster{
			Name:   cluster.GetName(),
			Client: k8sClient,
		}, nil

	}
	return nil, errors.New("no clusters found")
}
