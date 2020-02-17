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
	"encoding/json"
	"fmt"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/cmdutil"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"os"
	"strconv"
	"time"

	"github.com/gardener/test-infra/pkg/hostscheduler"
	flag "github.com/spf13/pflag"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

func (s *gkescheduler) List(flagset *flag.FlagSet) (hostscheduler.SchedulerFunc, error) {
	disableHeaders := flagset.Bool("no-headers", false, "Dont print headers when a table would be printed")
	outputType := flagset.StringP("output", "o", "", "Output format. One of json|yaml")
	return func(ctx context.Context) error {
		req := &containerpb.ListClustersRequest{
			Parent: s.getParentName(),
		}
		resp, err := s.client.ListClusters(ctx, req)
		if err != nil {
			return errors.Wrap(err, "unable to list gke clusters")
		}

		hosts := make([]*containerpb.Cluster, 0)
		for _, cluster := range resp.Clusters {
			if isHost(cluster.GetResourceLabels()) {
				hosts = append(hosts, cluster)
			}
		}

		if *outputType == "" {
			printTable(s.log, hosts, *disableHeaders)
			return nil
		}
		if *outputType == "json" {
			printJson(s.log, hosts)
			return nil
		}
		if *outputType == "yaml" {
			printYaml(s.log, hosts)
			return nil
		}

		return nil
	}, nil
}

func printYaml(log logr.Logger, clusters []*containerpb.Cluster) {
	fmt.Println(util.PrettyPrintStruct(clusters))
}

func printJson(log logr.Logger, clusters []*containerpb.Cluster) {
	dat, err := json.MarshalIndent(clusters, "", "  ")
	if err != nil {
		log.Error(err, "cannot marshal list")
		return
	}
	fmt.Print(string(dat))
}

func printTable(log logr.Logger, clusters []*containerpb.Cluster, disableHeaders bool) {
	headers := []string{"NAME", "STATUS", "ID", "LOCKED"}
	content := make([][]string, 0)

	for _, cluster := range clusters {
		labels := cluster.GetResourceLabels()
		ts := parseTimestamp(log, labels[ClusterLockedAtLabel])
		tsString := ""
		if !ts.IsZero() {
			tsString = ts.Format(time.RFC3339)
		}
		content = append(content, []string{
			cluster.GetName(),
			labels[ClusterStatusLabel],
			labels[ClusterLabelID],
			tsString,
		})
	}

	if disableHeaders {
		headers = make([]string, 0)
	}
	cmdutil.PrintTable(os.Stdout, headers, content)
}

func parseTimestamp(log logr.Logger, ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}
	i, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		log.V(3).Info(err.Error())
		return time.Time{}
	}

	return time.Unix(i, 0).UTC()
}
