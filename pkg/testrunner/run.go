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

package testrunner

import (
	"context"
	"errors"
	"fmt"
	"github.com/gardener/test-infra/pkg/testmachinery"
	trerrors "github.com/gardener/test-infra/pkg/testrunner/error"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"k8s.io/api/extensions/v1beta1"
	"net/url"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
	"k8s.io/apimachinery/pkg/util/wait"
)

// GetTestruns returns all testruns of a RunList as testrun array
func (rl RunList) GetTestruns() []*tmv1beta1.Testrun {
	testruns := make([]*tmv1beta1.Testrun, len(rl))
	for i, run := range rl {
		if run != nil {
			testruns[i] = run.Testrun
		}
	}
	return testruns
}

// HasErrors checks whether one run in list is erroneous.
func (rl RunList) HasErrors() bool {
	for _, run := range rl {
		if run.Error != nil {
			return true
		}
	}
	return false
}

// Errors returns all errors of all testruns in this testrun
func (rl RunList) Errors() error {
	var res *multierror.Error
	for _, run := range rl {
		if run.Error != nil {
			res = multierror.Append(res, run.Error)
		}
	}
	return util.ReturnMultiError(res)
}

func runTestrun(log logr.Logger, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun, namespace, name string) (*tmv1beta1.Testrun, error) {
	ctx := context.Background()
	defer ctx.Done()
	// TODO: Remove legacy name attribute. Instead enforce usage of generateName.
	tr.Name = ""
	tr.GenerateName = name
	tr.Namespace = namespace
	err := tmClient.Client().Create(ctx, tr)
	if err != nil {
		return nil, trerrors.NewNotCreatedError(fmt.Sprintf("cannot create testrun: %s", err.Error()))
	}
	log.Info(fmt.Sprintf("Testrun %s deployed", tr.Name))

	if argoUrl, err := GetArgoURL(tmClient, tr); err == nil {
		log.WithValues("testrun", tr.Name).Info(fmt.Sprintf("Argo workflow: %s", argoUrl))
	}

	testrunPhase := tmv1beta1.PhaseStatusInit
	err = wait.PollImmediate(pollInterval, maxWaitTime, func() (bool, error) {
		testrun := &tmv1beta1.Testrun{}
		err := tmClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: tr.Name}, testrun)
		if err != nil {
			log.Error(err, "cannot get testrun")
			return false, nil
		}
		tr = testrun

		if tr.Status.State != "" {
			testrunPhase = tr.Status.Phase
			log.Info(fmt.Sprintf("Testrun %s is in %s phase. State: %s", tr.Name, testrunPhase, tr.Status.State))
		} else {
			log.Info(fmt.Sprintf("Testrun %s is in %s phase. Waiting ...", tr.Name, testrunPhase))
		}
		return util.Completed(testrunPhase), nil
	})
	if err != nil {
		return nil, trerrors.NewTimeoutError(fmt.Sprintf("maximum wait time of %d is exceeded by Testrun %s", maxWaitTime, name))
	}

	return tr, nil
}

func GetArgoURL(tmClient kubernetes.Interface, tr *tmv1beta1.Testrun) (string, error) {
	// get argo url from the argo ingress if possible
	// return err if the ingress cannot be found
	argoIngress := &v1beta1.Ingress{}
	err := tmClient.Client().Get(context.TODO(), client.ObjectKey{Name: "argo-ui", Namespace: "default"}, argoIngress)
	if err != nil {
		return "", err
	}

	if len(argoIngress.Spec.Rules) == 0 {
		return "", errors.New("no rules defined")
	}
	rule := argoIngress.Spec.Rules[0]
	if len(rule.HTTP.Paths) == 0 {
		return "", errors.New("no http backend defined")
	}

	protocol := "http://"
	if len(argoIngress.Spec.TLS) != 0 {
		protocol = "https://"
	}
	argoUrl, err := url.ParseRequestURI(protocol + rule.Host)
	if err != nil {
		return "", err
	}
	argoUrl.Path = path.Join(argoUrl.Path, "workflows", tr.Namespace, testmachinery.GetWorkflowName(tr))

	return argoUrl.String(), nil
}
