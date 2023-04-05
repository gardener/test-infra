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

package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/go-logr/logr"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type testrunValidator struct {
	log     logr.Logger
	decoder *admission.Decoder
}

func NewValidator(log logr.Logger) admission.Handler {
	return &testrunValidator{
		log: log,
	}
}

// InjectDecoder injects the decoder.
// A decoder will be automatically injected.
func (v *testrunValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle validates a testrun
func (v *testrunValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	healthMutex.Lock()
	if healthErr != nil {
		defer healthMutex.Unlock()
		return admission.Errored(http.StatusFailedDependency, healthErr)
	}
	healthMutex.Unlock()

	tr := &tmv1beta1.Testrun{}
	if err := v.decoder.Decode(req, tr); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case admissionv1.Create:
		if err := validation.ValidateTestrun(tr); err != nil {
			v.log.V(5).Info(fmt.Sprintf("invalid testrun %s: %s", tr.Name, err.Error()))
			v.log.V(7).Info(string(req.Object.Raw))
			return admission.Denied(err.Error())
		}
	case admissionv1.Update:
		// forbid any update to the spec after the testrun was created
		oldObj := &tmv1beta1.Testrun{}
		if err := v.decoder.DecodeRaw(req.OldObject, oldObj); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if !reflect.DeepEqual(oldObj.Spec, tr.Spec) {
			v.log.V(5).Info(fmt.Sprintf("updated testrun spec %s", tr.Name))
			v.log.V(7).Info(string(req.Object.Raw))
			return admission.Denied("testrun spec is not allowed to be updated")
		}
	default:
		v.log.V(5).Info("Webhook not responsible")
	}

	return admission.Allowed("")
}
