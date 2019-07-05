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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/cmd/hostscheduler/scheduler"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/utils/retry"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"

	"cloud.google.com/go/container/apiv1"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

// WaitUntilShootIsReconciled waits until a cluster is reconciled and ready to use
func waitUntilOperationFinished(ctx context.Context, client *container.ClusterManagerClient, name string) (*containerpb.Operation, error) {
	newOperation := &containerpb.Operation{}
	interval := 15 * time.Second
	timeout := 30 * time.Minute
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		var err error
		opReq := &containerpb.GetOperationRequest{
			Name: name,
		}
		newOperation, err := client.GetOperation(ctx, opReq)
		if err != nil {
			return retry.MinorError(err)
		}

		log.Debugf("Operation %s is %s...", newOperation.GetName(), containerpb.Operation_Status_name[int32(newOperation.GetStatus())])

		if newOperation.Status == containerpb.Operation_DONE || newOperation.Status == containerpb.Operation_ABORTING {
			return retry.Ok()
		}
		return retry.NotOk()
	})
	if err != nil {
		return nil, err
	}
	return newOperation, nil
}

func clientFromCluster(cluster *containerpb.Cluster) (kubernetes.Interface, error) {
	auth := cluster.GetMasterAuth()

	ca, err := base64.StdEncoding.DecodeString(auth.GetClusterCaCertificate())
	if err != nil {
		return nil, err
	}
	cert, err := base64.StdEncoding.DecodeString(auth.GetClientCertificate())
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(auth.GetClientKey())
	if err != nil {
		return nil, err
	}

	cfg := &rest.Config{
		Host: cluster.GetEndpoint(),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: false,
			CAData:   ca,
			CertData: cert,
			KeyData:  key,
		},
		Username: auth.GetUsername(),
		Password: auth.GetPassword(),
	}

	return kubernetes.NewForConfig(cfg, client.Options{
		Scheme: kubernetes.ShootScheme,
	})
}

func (s gkescheduler) waitUntilOperationFinishedSuccessfully(ctx context.Context, operation *containerpb.Operation) error {
	operation, err := waitUntilOperationFinished(ctx, s.client, s.getOperationName(operation.GetName()))
	if err != nil {
		return err
	}
	if operation.Status == containerpb.Operation_ABORTING {
		return fmt.Errorf("Operation was aborted: %s", operation.GetStatusMessage())
	}
	return nil
}

func writeHostInformationToFile(host *hostCluster) error {
	hostConfig := config{
		Name: host.Name,
	}
	data, err := json.Marshal(hostConfig)
	if err != nil {
		log.Fatalf("cannot unmashal hostconfig: %s", err.Error())
	}

	err = os.MkdirAll(filepath.Dir(scheduler.HostConfigFilePath()), os.ModePerm)
	if err != nil {
		log.Fatalf("cannot create folder %s for host config: %s", filepath.Dir(scheduler.HostConfigFilePath()), err.Error())
	}
	err = ioutil.WriteFile(scheduler.HostConfigFilePath(), data, os.ModePerm)
	if err != nil {
		log.Fatalf("cannot write host config to %s: %s", scheduler.HostConfigFilePath(), err.Error())
	}

	return nil
}

func readHostInformationFromFile() (string, error) {
	dat, err := ioutil.ReadFile(scheduler.HostConfigFilePath())
	if err != nil {
		return "", err
	}

	c := &config{}
	err = json.Unmarshal(dat, c)
	if err != nil {
		return "", err
	}
	return c.Name, err
}
