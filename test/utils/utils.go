// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/retry"
	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
	kutil "github.com/gardener/test-infra/pkg/util/kubernetes"
)

// RunTestrunUntilCompleted executes a testrun on a cluster until it is finished and returns the corresponding executed testrun and workflow.
// Note: Deletion of the workflow on error should be handled by the calling test.
// DEPRECATED: this is just a wrapper function to keep functionality across tests. In the future the RunTestrun function should be directly used.
func RunTestrunUntilCompleted(ctx context.Context, log logr.Logger, tmClient client.Client, tr *tmv1beta1.Testrun, phase argov1.WorkflowPhase, timeout time.Duration) (*tmv1beta1.Testrun, *argov1.Workflow, error) {
	return RunTestrun(ctx, log, tmClient, tr, phase, timeout, WatchUntilCompletedFunc)
}

// RunTestrunUntilCompleted executes a testrun on a cluster until a specific state has been reached and returns the corresponding executed testrun and workflow.
// Note: Deletion of the workflow on error should be handled by the calling test.
func RunTestrun(ctx context.Context, log logr.Logger, tmClient client.Client, tr *tmv1beta1.Testrun, phase argov1.WorkflowPhase, timeout time.Duration, f WatchFunc) (*tmv1beta1.Testrun, *argov1.Workflow, error) {
	err := tmClient.Create(ctx, tr)
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
func GetWorkflow(ctx context.Context, tmClient client.Client, tr *tmv1beta1.Testrun) (*argov1.Workflow, error) {
	wf := &argov1.Workflow{}
	err := tmClient.Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Status.Workflow}, wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

// WatchFunc is the function to check if the testrun has a specific state
type WatchFunc = func(*tmv1beta1.Testrun) (bool, error)

// WatchTestrun watches a testrun until the maxwait is reached.
// Every time a testrun can be found without a problem the function f is called.
func WatchTestrun(ctx context.Context, tmClient client.Client, tr *tmv1beta1.Testrun, timeout time.Duration, f WatchFunc) error {
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, timeout, true, func(ctx context.Context) (bool, error) {
		updatedTestrun := &tmv1beta1.Testrun{}
		if err := tmClient.Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, updatedTestrun); err != nil {
			return retry.MinorError(err)
		}
		*tr = *updatedTestrun
		return f(updatedTestrun)
	})
}

// WatchUntilCompletedFunc waits until the testrun is completed
func WatchUntilCompletedFunc(tr *tmv1beta1.Testrun) (bool, error) {
	testrunPhase := tr.Status.Phase
	if util.CompletedRun(testrunPhase) {
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
func WatchTestrunUntilCompleted(ctx context.Context, tmClient client.Client, tr *tmv1beta1.Testrun, timeout time.Duration) (*tmv1beta1.Testrun, error) {
	foundTestrun := tr.DeepCopy()
	err := WatchTestrun(ctx, tmClient, tr, timeout, WatchUntilCompletedFunc)
	return foundTestrun, err
}

// DeleteTestrun deletes a testrun and expects to be successful.
func DeleteTestrun(tmClient client.Client, tr *tmv1beta1.Testrun) {
	// wf is not deleted if testrun is triggered but deleted before wf can be deployed.
	// Strange timing in validation test with kubeconfig.
	// needs further investigation
	time.Sleep(3 * time.Second)
	err := tmClient.Delete(context.TODO(), tr)
	if !apierrors.IsNotFound(err) {
		Expect(err).To(BeNil(), "Error deleting Testrun: %s", tr.Name)
	}
}

// DumpState dumps the current testrun und workflow state
func DumpState(ctx context.Context, log logr.Logger, client client.Client, tr, foundTestrun *tmv1beta1.Testrun) {
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
func WaitForClusterReadiness(log logr.Logger, clusterClient client.Client, namespace string, maxWaitTime time.Duration) error {
	ctx := context.Background()
	defer ctx.Done()
	return wait.PollUntilContextTimeout(ctx, 5*time.Second, maxWaitTime, true, func(context context.Context) (bool, error) {
		var (
			tmControllerStatus = deploymentIsReady(ctx, log, clusterClient, namespace, "testmachinery-controller")
		)
		if tmControllerStatus {
			return true, nil
		}
		log.Info("waiting for Test Machinery components to become ready", "TestMachinery-controller", tmControllerStatus)
		return false, nil
	})
}

func deploymentIsReady(ctx context.Context, log logr.Logger, clusterClient client.Client, namespace, name string) bool {
	deployment := &appsv1.Deployment{}
	err := clusterClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, deployment)
	if err != nil {
		log.V(3).Info(err.Error())
		return false
	}
	err = kutil.CheckDeployment(deployment)
	return err == nil
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

// ReadJSONFile reads a file and deserializes the json into the given object
func ReadJSONFile(path string, obj interface{}) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

// ReadYAMLFile reads a file and deserializes the yaml into the given object
func ReadYAMLFile(path string, obj interface{}) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, obj)
}
