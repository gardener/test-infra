// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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

// TODO: refactor testrunValidator corresponding to the controller-runtime changes from v0.14 to v0.15 (see
// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.15.0/examples/builtins/validatingwebhook.go)
func NewValidatorWithDecoder(log logr.Logger, decoder *admission.Decoder) admission.Handler {
	return &testrunValidator{
		log:     log,
		decoder: decoder,
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
