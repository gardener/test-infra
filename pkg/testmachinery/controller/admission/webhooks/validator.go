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

package webhooks

import (
	"context"
	"fmt"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/testrun"
	"github.com/go-logr/logr"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

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
	tr := &tmv1beta1.Testrun{}

	if err := v.decoder.Decode(req, tr); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case admissionv1beta1.Create:
		if err := testrun.Validate(v.log.WithValues("testrun", tr.Name), tr); err != nil {
			v.log.V(5).Info(fmt.Sprintf("invalid testrun %s: %s", tr.Name, err.Error()))
			return admission.Denied(err.Error())
		}
	default:
		v.log.V(5).Info("Webhook not responsible")
	}

	return admission.Allowed("")
}
