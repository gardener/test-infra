// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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
	"github.com/gardener/gardener-resource-manager/pkg/apis/resources/v1alpha1"
	mrhealth "github.com/gardener/gardener-resource-manager/pkg/health"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"reflect"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/utils/kubernetes/health"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	"github.com/gardener/test-infra/pkg/util"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	. "github.com/onsi/gomega"
)

// RunTestrunUntilCompleted executes a testrun on a cluster until it is finished and returns the corresponding executed testrun and workflow.
// Note: Deletion of the workflow on error should be handled by the calling test.
// DEPRECATED: this is just a wrapper function to keep functionality across tests. In the future the RunTestrun function should be directly used.
func RunTestrunUntilCompleted(ctx context.Context, log logr.Logger, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, phase argov1.NodePhase, timeout time.Duration) (*tmv1beta1.Testrun, *argov1.Workflow, error) {
	return RunTestrun(ctx, log, tmClient, tr, phase, timeout, WatchUntilCompletedFunc)
}

// RunTestrunUntilCompleted executes a testrun on a cluster until a specific state has been reached and returns the corresponding executed testrun and workflow.
// Note: Deletion of the workflow on error should be handled by the calling test.
func RunTestrun(ctx context.Context, log logr.Logger, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, phase argov1.NodePhase, timeout time.Duration, f WatchFunc) (*tmv1beta1.Testrun, *argov1.Workflow, error) {
	err := tmClient.Client().Create(ctx, tr)
	if err != nil {
		return nil, nil, err
	}

	foundTestrun := tr.DeepCopy()
	err = WatchTestrun(ctx, tmClient, foundTestrun, timeout, f)
	if err != nil {
		DumpState(ctx, log, tmClient, tr, foundTestrun)
		return nil, nil, errors.Wrapf(err, "error watching Testrun %s in Namespace %s", tr.Name, tr.Namespace)
	}

	wf, err := GetWorkflow(ctx, tmClient, foundTestrun)
	if err != nil {
		DumpState(ctx, log, tmClient, tr, foundTestrun)
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

// WatchFunc is the function to check if the testrun has a specific state
type WatchFunc = func(*tmv1beta1.Testrun) (bool, error)

// WatchTestrun watches a testrun until the maxwait is reached.
// Every time a testrun can be found without a problem the function f is called.
func WatchTestrun(ctx context.Context, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, timeout time.Duration, f WatchFunc) error {
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		updatedTestrun := &tmv1beta1.Testrun{}
		if err := tmClient.Client().Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, updatedTestrun); err != nil {
			return retry.MinorError(err)
		}
		*tr = *updatedTestrun
		return f(updatedTestrun)
	})
}

// WatchUntilCompletedFunc waits until the testrun is completed
func WatchUntilCompletedFunc(tr *tmv1beta1.Testrun) (bool, error) {
	testrunPhase := tr.Status.Phase
	if util.Completed(testrunPhase) {
		return retry.Ok()
	}
	return retry.NotOk()
}

// WatchUntil waits a certain timeout before the testrun watch is canceled.
func WatchUntil(timeout time.Duration) WatchFunc {
	finished := false
	timer := time.NewTimer(timeout)
	go func() {
		<-timer.C
		finished = true
	}()
	return func(tr *tmv1beta1.Testrun) (bool, error) {
		if finished {
			return retry.Ok()
		}
		return retry.NotOk()
	}
}

// WatchTestrunUntilCompleted watches a testrun to finish and returns the newest testrun object.
func WatchTestrunUntilCompleted(ctx context.Context, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, timeout time.Duration) (*tmv1beta1.Testrun, error) {
	foundTestrun := tr.DeepCopy()
	err := WatchTestrun(ctx, tmClient, tr, timeout, WatchUntilCompletedFunc)
	return foundTestrun, err
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

// DumpState dumps the current testrun und workflow state
func DumpState(ctx context.Context, log logr.Logger, client kubernetes.Interface, tr, foundTestrun *tmv1beta1.Testrun) {
	fmt.Println("Testrun:")
	fmt.Println(util.PrettyPrintStruct(tr))
	fmt.Println("FoundTestrun:")
	fmt.Println(util.PrettyPrintStruct(foundTestrun))

	// try to get the workflow and dump its error message
	wf, err := GetWorkflow(ctx, client, foundTestrun)
	if err != nil {
		log.Info("unable to get workflow for testrun", "error", err.Error(), "workflow", foundTestrun.Status.Workflow)
		return
	}
	fmt.Printf("Argo Workflow %s (Phase %s): %s", wf.Name, wf.Status.Phase, wf.Status.Message)
}

// WaitForClusterReadiness waits for all testmachinery components to be ready.
func WaitForClusterReadiness(log logr.Logger, clusterClient kubernetes.Interface, namespace string, maxWaitTime time.Duration) error {
	ctx := context.Background()
	defer ctx.Done()
	return wait.PollImmediate(5*time.Second, maxWaitTime, func() (bool, error) {
		var (
			tmControllerStatus = deploymentIsReady(ctx, log, clusterClient, namespace, "testmachinery-controller")
			argoStatus         = managedresourceIsReady(ctx, log, clusterClient, namespace, config.ArgoManagedResourceName)
			minioStatus        = managedresourceIsReady(ctx, log, clusterClient, namespace, config.MinioManagedResourceName)
		)
		if tmControllerStatus && argoStatus && minioStatus {
			return true, nil
		}
		log.Info("waiting for Test Machinery components to become ready", "TestMachinery-controller", tmControllerStatus, "argo", argoStatus, "minio", minioStatus)
		return false, nil
	})
}

// WaitForMinioService waits for the minio service to get an external IP and return the minio config.
func WaitForMinioService(minioEndpoint string, maxWaitTime time.Duration) error {
	// wait for service to get endpoint ip
	return wait.PollImmediate(10*time.Second, maxWaitTime, func() (bool, error) {
		_, err := HTTPGet("http://" + minioEndpoint)
		if err != nil {
			return retry.MinorError(err)
		}

		return retry.Ok()
	})
}

func deploymentIsReady(ctx context.Context, log logr.Logger, clusterClient kubernetes.Interface, namespace, name string) bool {
	deployment := &appsv1.Deployment{}
	err := clusterClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment)
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

func managedresourceIsReady(ctx context.Context, log logr.Logger, clusterClient kubernetes.Interface, namespace, name string) bool {
	mr := &v1alpha1.ManagedResource{}
	err := clusterClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, mr)
	if err != nil {
		log.V(3).Info(err.Error())
		return false
	}
	err = mrhealth.CheckManagedResource(mr)
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
