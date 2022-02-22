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

	container "cloud.google.com/go/container/apiv1"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/client-go/rest"

	"github.com/gardener/test-infra/pkg/hostscheduler"

	"github.com/gardener/gardener/pkg/utils/retry"
	"k8s.io/apimachinery/pkg/util/wait"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
)

// WaitUntilShootIsReconciled waits until a cluster is reconciled and ready to use
func waitUntilOperationFinished(log logr.Logger, ctx context.Context, client *container.ClusterManagerClient, name string) (*containerpb.Operation, error) {
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

		log.Info(fmt.Sprintf("Operation %s is %s...", newOperation.GetName(), containerpb.Operation_Status_name[int32(newOperation.GetStatus())]))

		if newOperation.Status == containerpb.Operation_DONE || newOperation.Status == containerpb.Operation_ABORTING {
			return retry.Ok()
		}
		return retry.NotOk()
	})
	if err != nil {
		return nil, errors.Wrapf(err, "error while waiting for operation %s to finish", name)
	}
	return newOperation, nil
}

func clientFromCluster(cluster *containerpb.Cluster) (client.Client, error) {
	restConfig, err := restConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}
	return client.New(restConfig, client.Options{})
}

func restConfigFromCluster(cluster *containerpb.Cluster) (*rest.Config, error) {
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
		Username: auth.GetUsername(), //nolint
		Password: auth.GetPassword(), //nolint
	}
	return cfg, nil
}

func (s *gkescheduler) waitUntilOperationFinishedSuccessfully(ctx context.Context, operation *containerpb.Operation) error {
	operation, err := waitUntilOperationFinished(s.log, ctx, s.client, s.getOperationName(operation.GetName()))
	if err != nil {
		return err
	}
	if operation.Status == containerpb.Operation_ABORTING {
		return fmt.Errorf("operation was aborted: %s", operation.GetError().Message)
	}
	return nil
}

func writeHostInformationToFile(hostName string) error {
	hostConfigPath, err := hostscheduler.HostConfigFilePath()
	if err != nil {
		return nil
	}

	hostConfig := config{
		Name: hostName,
	}
	data, err := json.Marshal(hostConfig)
	if err != nil {
		return errors.Wrapf(err, "cannot unmashal hostconfig")
	}

	err = os.MkdirAll(filepath.Dir(hostConfigPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot create folder %s for host config: %s", filepath.Dir(hostConfigPath), err.Error())
	}
	err = ioutil.WriteFile(hostConfigPath, data, os.ModePerm)
	if err != nil {
		return fmt.Errorf("cannot write host config to %s: %s", hostConfigPath, err.Error())
	}

	return nil
}

func readHostInformationFromFile() (string, error) {
	hostConfigPath, err := hostscheduler.HostConfigFilePath()
	if err != nil {
		return "", err
	}
	dat, err := ioutil.ReadFile(hostConfigPath)
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
