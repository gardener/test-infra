// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/utils/kubernetes/health"

	"github.com/gardener/test-infra/pkg/testmachinery"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	"github.com/gardener/test-infra/pkg/util"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	. "github.com/onsi/gomega"
)

// RunTestrun executes a testrun on a cluster and returns the corresponding executed testrun and workflow.
// Note: Deletion of the workflow on error should be handled by the calling test.
func RunTestrun(ctx context.Context, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, phase argov1.NodePhase, namespace string, maxWaitTime int64) (*tmv1beta1.Testrun, *argov1.Workflow, error) {
	err := tmClient.Client().Create(ctx, tr)
	if err != nil {
		return nil, nil, err
	}

	foundTestrun, err := WatchTestrun(ctx, tmClient, tr, namespace, maxWaitTime)
	if err != nil {
		return nil, nil, fmt.Errorf("Error watching Testrun: %s\n%s", tr.Name, err.Error())
	}

	wf, err := GetWorkflow(ctx, tmClient, foundTestrun)
	if err != nil {
		return nil, nil, fmt.Errorf("Cannot get Workflow for Testrun: %s\n%s", tr.Name, err.Error())
	}

	if reflect.DeepEqual(foundTestrun.Status, tmv1beta1.TestrunStatus{}) {
		return nil, nil, fmt.Errorf("Testrun %s status is empty", tr.Name)
	}
	if foundTestrun.Status.Phase != phase {
		// get additional errors message
		errMsgs := []string{}
		for _, node := range wf.Status.Nodes {
			if node.Message != "" {
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %s", node.TemplateName, node.Message))
			}
		}

		errMsg := fmt.Sprintf("Testrun %s status should be %s, but is %s", tr.Name, phase, foundTestrun.Status.Phase)
		if wf.Status.Message != "" {
			errMsg = fmt.Sprintf("%s. Workflow Message: %s", errMsg, wf.Status.Message)
		}
		if len(errMsgs) != 0 {
			errMsg = fmt.Sprintf("%s.\nAdditional Errors: %s", errMsg, strings.Join(errMsgs, "; "))
		}

		return nil, nil, errors.New(errMsg)
	}

	return foundTestrun, wf, nil
}

// GetWorkflow returns the argo workflow of a testrun.
func GetWorkflow(ctx context.Context, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun) (*argov1.Workflow, error) {
	wf := &argov1.Workflow{}
	err := tmClient.Client().Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Status.Workflow}, wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

// WatchTestrun watches a testrun to finish and returns the newest testrun object.
func WatchTestrun(ctx context.Context, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, namespace string, maxWaitTime int64) (*tmv1beta1.Testrun, error) {
	foundTestrun := &tmv1beta1.Testrun{}
	var testrunPhase argov1.NodePhase
	startTime := time.Now()
	for !util.Completed(testrunPhase) {
		var err error
		if util.MaxTimeExceeded(startTime, maxWaitTime) {
			return nil, fmt.Errorf("Maximum wait time exceeded")
		}

		err = tmClient.Client().Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, foundTestrun)
		if err != nil {
			return nil, err
		}
		testrunPhase = foundTestrun.Status.Phase

		time.Sleep(5 * time.Second)
	}

	return foundTestrun, nil
}

// DeleteTestrun deletes a testrun and expects to be successful.
func DeleteTestrun(tmClient kubernetes.Interface, tr *tmv1beta1.Testrun) {
	// wf is not deleted if testrun is triggered but deleted before wf can be deployed.
	// Strange timing in validation test with kubeconfig.
	// needs further investigation
	time.Sleep(3 * time.Second)
	err := tmClient.Client().Delete(context.TODO(), tr)
	if !apierrors.IsNotFound(err) {
		Expect(err).To(BeNil(), "Error deleting Testrun: %s", tr.Name)
	}
}

// WaitForClusterReadiness waits for all testmachinery components to be ready.
func WaitForClusterReadiness(clusterClient kubernetes.Interface, namespace string, maxWaitTime int64) {
	startTime := time.Now()
	for {
		Expect(util.MaxTimeExceeded(startTime, maxWaitTime)).To(BeFalse(), "Max Wait time for cluster readiness exceeded.")
		if deploymentIsReady(clusterClient, namespace, "testmachinery-controller") &&
			deploymentIsReady(clusterClient, namespace, "workflow-controller") &&
			deploymentIsReady(clusterClient, namespace, "minio-deployment") {
			break
		}
	}
}

// WaitForMinioService waits for the minio service to get an external IP and return the minio config.
func WaitForMinioService(clusterClient kubernetes.Interface, minioEndpoint, namespace string, maxWaitTime int64) *testmachinery.ObjectStoreConfig {
	minioConfig, err := clusterClient.GetConfigMap(namespace, "tm-config")
	Expect(err).ToNot(HaveOccurred())

	minioSecrets, err := clusterClient.GetSecret(namespace, minioConfig.Data["objectstore.secretName"])
	Expect(err).ToNot(HaveOccurred())

	// wait for service to get endpoint ip
	if minioEndpoint != "" {
		startTime := time.Now()
		for {
			Expect(util.MaxTimeExceeded(startTime, maxWaitTime)).To(BeFalse(), "Max Wait time for minio external ip exceeded.")
			_, err := HTTPGet("http://" + minioEndpoint)
			if err == nil {
				break
			}
		}
	}

	return &testmachinery.ObjectStoreConfig{
		Endpoint:   minioEndpoint,
		AccessKey:  string(minioSecrets.Data["accessKey"]),
		SecretKey:  string(minioSecrets.Data["secretKey"]),
		BucketName: minioConfig.Data["objectstore.bucketName"],
	}
}

func deploymentIsReady(clusterClient kubernetes.Interface, namespace, name string) bool {
	deployment, err := clusterClient.GetDeployment(namespace, name)
	if err != nil {
		return false
	}
	err = health.CheckDeployment(deployment)
	if err == nil {
		return true
	}
	return false
}

// HTTPGet performs an HTTP get with a default timeout of 60 seconds
func HTTPGet(url string) (*http.Response, error) {
	httpClient := http.Client{
		Timeout: time.Duration(60 * time.Second),
	}

	httpRequest, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// TestflowLen returns the number of all items in 2 dimensional array.
func TestflowLen(m [][]*tmv1beta1.TestflowStepStatus) int {
	length := 0
	for _, a := range m {
		length += len(a)
	}
	return length
}
