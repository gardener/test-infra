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

	"cloud.google.com/go/container/apiv1/containerpb"
	"github.com/pkg/errors"
	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/gardener/test-infra/pkg/hostscheduler"
	"github.com/gardener/test-infra/pkg/hostscheduler/cleanup"
)

var (
	NotHeapster          = cleanup.MustNewRequirement("k8s-app", selection.NotEquals, "heapster")
	NotKubeDNS           = cleanup.MustNewRequirement("k8s-app", selection.NotEquals, "kube-dns")
	NotKubeDNSAutoscaler = cleanup.MustNewRequirement("k8s-app", selection.NotEquals, "kube-dns-autoscaler")
	NotMetricsServer     = cleanup.MustNewRequirement("k8s-app", selection.NotEquals, "metrics-server")
	NotKubeProxy         = cleanup.MustNewRequirement("component", selection.NotEquals, "kube-proxy")
)

func (s *gkescheduler) Cleanup(flagset *flag.FlagSet) (hostscheduler.SchedulerFunc, error) {
	clean := flagset.Bool("clean", false, "Cleanup the specified cluster")

	return func(ctx context.Context) error {
		if clean != nil && !*clean {
			s.log.V(3).Info("clean is not defined. Therefore the cluster is not cleaned up.")
			return nil
		}
		if s.hostname == "" {
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
			return errors.Wrap(err, "unable to fetch cluster")
		}
		s.log.Info("starting to cleanup cluster")

		k8sClient, err := clientFromCluster(cluster)
		if err != nil {
			return errors.Wrap(err, "unable to build kubernetes client")
		}

		if err := cleanup.CleanResources(ctx, s.log, k8sClient, labels.Requirements{NotHeapster, NotKubeDNS, NotKubeDNSAutoscaler, NotMetricsServer, NotKubeProxy}); err != nil {
			return errors.Wrap(err, "unable to cleanup cluster")
		}
		return nil
	}, nil
}
