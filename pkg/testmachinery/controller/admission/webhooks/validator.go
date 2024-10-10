// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package webhooks

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1/validation"
	kutil "github.com/gardener/test-infra/pkg/util/kubernetes"
)

var (
	healthMutex sync.Mutex
	healthErr   error
)

// StartHealthCheck will start a go routine periodically checking the health of a specified deployment
// The result of the checks is available to the testRunValidator only
func StartHealthCheck(ctx context.Context, reader client.Reader, namespace string, deploymentName string, interval metav1.Duration) {
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			checkDeploymentHealth(ctx, reader, namespace, deploymentName)
			time.Sleep(interval.Duration)
		}
	}()
}

func checkDeploymentHealth(ctx context.Context, reader client.Reader, namespace string, deploymentName string) {
	deployment := &appsv1.Deployment{}
	err := reader.Get(ctx, client.ObjectKey{Name: deploymentName, Namespace: namespace}, deployment)
	if err != nil {
		healthMutex.Lock()
		defer healthMutex.Unlock()
		healthErr = err
		return
	}

	err = kutil.CheckDeployment(deployment)
	healthMutex.Lock()
	defer healthMutex.Unlock()
	healthErr = err
}

type TestRunCustomValidator struct {
	Log logr.Logger
}

func (v *TestRunCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	// permit new TestRuns only when the dependency health check is successful
	healthMutex.Lock()
	if healthErr != nil {
		defer healthMutex.Unlock()
		return nil, healthErr
	}
	healthMutex.Unlock()

	tr, ok := obj.(*tmv1beta1.Testrun)
	if !ok {
		return nil, fmt.Errorf("expected a TestRun object but got type: %T", obj)
	}
	if err := validation.ValidateTestrun(tr); err != nil {
		v.Log.V(5).Info(fmt.Sprintf("invalid testrun %s: %s", tr.Name, err.Error()))
		return nil, err
	}
	return nil, nil
}

func (v *TestRunCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (warnings admission.Warnings, err error) {
	newTr, ok := newObj.(*tmv1beta1.Testrun)
	if !ok {
		return nil, fmt.Errorf("expected a TestRun as new object but got type: %T", newObj)
	}

	oldTr, ok := oldObj.(*tmv1beta1.Testrun)
	if !ok {
		return nil, fmt.Errorf("expected a TestRun old object but got type: %T", oldObj)
	}

	if !reflect.DeepEqual(oldTr.Spec, newTr.Spec) {
		v.Log.V(5).Info(fmt.Sprintf("forbidden update of testrun spec for %s", newTr.Name))
		return nil, errors.NewInvalid(
			schema.GroupKind{
				Group: tmv1beta1.SchemeGroupVersion.Group,
				Kind:  newTr.GetObjectKind().GroupVersionKind().Kind},
			newTr.Name,
			field.ErrorList{
				field.Forbidden(field.NewPath("spec"), "testrun spec is not allowed to be updated"),
			},
		)
	}
	return nil, nil
}

func (v *TestRunCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	_, ok := obj.(*tmv1beta1.Testrun)
	if !ok {
		return admission.Warnings{}, fmt.Errorf("expected a TestRun object but got object type: %T", obj)
	}
	return nil, nil
}
