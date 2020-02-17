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
	"fmt"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"strconv"
	"time"
)

const (
	SelectHostTimeout = 2 * time.Hour
)

func (s *gkescheduler) Lock(flagset *flag.FlagSet) (hostscheduler.SchedulerFunc, error) {
	id := flagset.String("id", "", "Metadata representing a unique identifier for the cluster")

	return func(ctx context.Context) error {
		var (
			cluster   *containerpb.Cluster
			k8sClient kubernetes.Interface
			err       error
		)
		if s.hostname == "" {
			cluster, k8sClient, err = s.selectAvailableHost(ctx)
			if err != nil {
				return errors.Wrap(err, "unable to select host cluster")
			}
		} else {
			req := &containerpb.GetClusterRequest{
				Name: s.getClusterName(s.hostname),
			}
			cluster, err = s.client.GetCluster(ctx, req)
			if err != nil {
				return errors.Wrap(err, "unable to get gke clusters")
			}
			k8sClient, err = clientFromCluster(cluster)
			if err != nil {
				return errors.Wrap(err, "unable to build k8s client from cluster")
			}
		}
		s.log.Info(fmt.Sprintf("Selected %s", s.getClusterName(cluster.GetName())))
		s.log = s.log.WithValues("host", s.getClusterName(cluster.GetName()))

		s.log.Info("update labels of cluster")
		labels := cluster.GetResourceLabels()
		labels[ClusterStatusLabel] = ClusterStatusLocked
		labels[ClusterLockedAtLabel] = strconv.FormatInt(time.Now().Unix(), 10)
		if *id != "" {
			labels[ClusterLabelID] = *id
		}

		labelsRequest := &containerpb.SetLabelsRequest{
			Name:             s.getClusterName(cluster.GetName()),
			ResourceLabels:   labels,
			LabelFingerprint: cluster.LabelFingerprint,
		}
		o, err := s.client.SetLabels(ctx, labelsRequest)
		if err != nil {
			return errors.Wrap(err, "labels cannot be set")
		}

		if err := s.waitUntilOperationFinishedSuccessfully(ctx, o); err != nil {
			return errors.Wrap(err, "labels cannot be set")
		}

		if err := hostscheduler.WriteHostKubeconfig(s.log, k8sClient); err != nil {
			return errors.Wrap(err, "unable to write host kubeconfig")
		}

		return writeHostInformationToFile(cluster.GetName())
	}, nil
}

func (s *gkescheduler) selectAvailableHost(ctx context.Context) (*containerpb.Cluster, kubernetes.Interface, error) {
	var (
		cluster   *containerpb.Cluster
		k8sClient kubernetes.Interface
		err       error
	)
	err = retry.UntilTimeout(ctx, 1*time.Minute, SelectHostTimeout, func(ctx context.Context) (bool, error) {
		cluster, k8sClient, err = tryAvailableHost(ctx, s.log, s.client, s.getParentName())
		if err != nil {
			s.log.Info("Unable to select host cluster. Trying again..")
			s.log.V(3).Info(err.Error())
			return retry.MinorError(err)
		}
		return retry.Ok()
	})
	return cluster, k8sClient, err
}

func tryAvailableHost(ctx context.Context, logger logr.Logger, client *container.ClusterManagerClient, parent string) (*containerpb.Cluster, kubernetes.Interface, error) {
	req := &containerpb.ListClustersRequest{
		Parent: parent,
	}
	resp, err := client.ListClusters(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	for _, cluster := range resp.Clusters {
		if cluster.GetStatus() != containerpb.Cluster_RUNNING {
			logger.V(3).Info("found cluster but is unavailable", "host", cluster.Name, "state", containerpb.Cluster_Status_name[int32(cluster.GetStatus())])
			continue
		}
		labels := cluster.GetResourceLabels()
		if !isHost(labels) {
			logger.V(3).Info(fmt.Sprintf("found cluster but it has no host cluster with label %s", ClusterLabel), "host", cluster.GetName())
			continue
		}
		if isLocked(labels) {
			logger.V(3).Info("found cluster but it is locked", "host", cluster.GetName())
			continue
		}

		logger.Info(fmt.Sprintf("use gke cluster %s", cluster.GetName()))
		k8sClient, err := clientFromCluster(cluster)
		if err != nil {
			logger.V(3).Info(err.Error())
			continue
		}

		return cluster, k8sClient, nil

	}
	return nil, nil, errors.New("no clusters found")
}
