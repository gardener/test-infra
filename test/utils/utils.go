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
	"fmt"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
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
func RunTestrun(ctx context.Context, log logr.Logger, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, phase argov1.NodePhase, maxWaitTime time.Duration) (*tmv1beta1.Testrun, *argov1.Workflow, error) {
	err := tmClient.Client().Create(ctx, tr)
	if err != nil {
		return nil, nil, err
	}

	foundTestrun, err := WatchTestrunUntilCompleted(ctx, tmClient, tr, maxWaitTime)
	if err != nil {
		fmt.Println("Testrun:")
		fmt.Println(util.PrettyPrintStruct(tr))
		fmt.Println("FoundTestrun:")
		fmt.Println(util.PrettyPrintStruct(foundTestrun))
		return nil, nil, errors.Wrapf(err, "error watching Testrun %s in Namespace %s", tr.Name, tr.Namespace)
	}

	wf, err := GetWorkflow(ctx, tmClient, foundTestrun)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get Workflow for Testrun: %s: %s", tr.Name, err.Error())
	}

	if reflect.DeepEqual(foundTestrun.Status, tmv1beta1.TestrunStatus{}) {
		return nil, nil, errors.Wrapf(err, "Testrun %s in namespace % status is empty", tr.Name, tr.Namespace)
	}
	if foundTestrun.Status.Phase != phase {
		// get additional errors message
		errMsgs := make([]string, 0)
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

// WatchTestrunUntilCompleted watches a testrun to finish and returns the newest testrun object.
func WatchTestrunUntilCompleted(ctx context.Context, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, maxWaitTime time.Duration) (*tmv1beta1.Testrun, error) {
	foundTestrun := &tmv1beta1.Testrun{}
	var testrunPhase argov1.NodePhase

	err := WatchTestrun(ctx, tmClient, tr, maxWaitTime, func(newTestrun *tmv1beta1.Testrun) (bool, error) {
		foundTestrun = newTestrun
		testrunPhase = foundTestrun.Status.Phase
		if util.Completed(testrunPhase) {
			return retry.Ok()
		}
		return retry.NotOk()
	})

	return foundTestrun, err
}

// WatchTestrun watches a testrun until the maxwait is reached.
// Every time a testrun can be found without a problem the function f is called.
func WatchTestrun(ctx context.Context, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, maxWaitTime time.Duration, f func(*tmv1beta1.Testrun) (bool, error)) error {
	return wait.PollImmediate(5*time.Second, maxWaitTime, func() (bool, error) {
		updatedTestrun := &tmv1beta1.Testrun{}
		if err := tmClient.Client().Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, updatedTestrun); err != nil {
			return retry.MinorError(err)
		}
		return f(updatedTestrun)
	})
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
func WaitForClusterReadiness(log logr.Logger, clusterClient kubernetes.Interface, namespace string, maxWaitTime time.Duration) error {
	return wait.PollImmediate(5*time.Second, maxWaitTime, func() (bool, error) {
		var (
			tmControllerStatus    = deploymentIsReady(log, clusterClient, namespace, "testmachinery-controller")
			wfControllerStatus    = deploymentIsReady(log, clusterClient, namespace, "workflow-controller")
			minioDeploymentStatus = deploymentIsReady(log, clusterClient, namespace, "minio-deployment")
		)
		if tmControllerStatus && wfControllerStatus && minioDeploymentStatus {
			return true, nil
		}
		log.Info("waiting for Test Machinery components to become ready", "TestMachinery-controller", tmControllerStatus, "workflow-controller", wfControllerStatus, "minio", minioDeploymentStatus)
		return false, nil
	})
}

// WaitForMinioService waits for the minio service to get an external IP and return the minio config.
func WaitForMinioService(clusterClient kubernetes.Interface, minioEndpoint, namespace string, maxWaitTime time.Duration) (*testmachinery.S3Config, error) {
	ctx := context.Background()
	defer ctx.Done()

	minioConfig := &corev1.ConfigMap{}
	err := clusterClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: "tm-config"}, minioConfig)
	Expect(err).ToNot(HaveOccurred())

	minioSecret := &corev1.Secret{}
	err = clusterClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: minioConfig.Data["objectstore.secretName"]}, minioSecret)
	Expect(err).ToNot(HaveOccurred())

	// wait for service to get endpoint ip
	err = wait.PollImmediate(10*time.Second, maxWaitTime, func() (bool, error) {
		_, err := HTTPGet("http://" + minioEndpoint)
		if err != nil {
			return retry.MinorError(err)
		}

		return retry.Ok()
	})
	if err != nil {
		return nil, err
	}

	return &testmachinery.S3Config{
		Endpoint:   minioEndpoint,
		AccessKey:  string(minioSecret.Data["accessKey"]),
		SecretKey:  string(minioSecret.Data["secretKey"]),
		BucketName: minioConfig.Data["objectstore.bucketName"],
	}, nil
}

func deploymentIsReady(log logr.Logger, clusterClient kubernetes.Interface, namespace, name string) bool {
	deployment := &appsv1.Deployment{}
	err := clusterClient.Client().Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, deployment)
	if err != nil {
		log.V(3).Info(err.Error())
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
