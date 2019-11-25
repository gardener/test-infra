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
	"net/url"
	"path"
	"regexp"

	"github.com/gardener/test-infra/pkg/testmachinery"
	trerrors "github.com/gardener/test-infra/pkg/testrunner/error"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-multierror"
	"k8s.io/api/extensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
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

func (r *Run) Exec(log logr.Logger, config *Config, prefix string) {
	ctx := context.Background()
	defer ctx.Done()
	newTR := r.Testrun.DeepCopy()

	// Remove legacy name attribute. Instead enforce usage of generateName.
	newTR.Name = ""
	newTR.GenerateName = prefix
	newTR.Namespace = config.Namespace
	err := config.Watch.Client().Create(ctx, newTR)
	if err != nil {
		log.Error(err, "unable to create testrun")
		r.Error = trerrors.NewNotCreatedError(fmt.Sprintf("cannot create testrun: %s", err.Error()))
		return
	}

	*r.Testrun = *newTR
	r.Metadata.Testrun.ID = newTR.GetName()
	log.Info(fmt.Sprintf("Testrun %s deployed", newTR.Name))

	if argoUrl, err := GetArgoURL(config.Watch.Client(), r.Testrun); err == nil {
		log.WithValues("testrun", r.Testrun.GetName()).Info(fmt.Sprintf("Argo workflow: %s", argoUrl))
	}

	testrunPhase := tmv1beta1.PhaseStatusInit
	err = config.Watch.WatchUntil(config.Timeout, r.Testrun.GetNamespace(), r.Testrun.GetName(), func(new *tmv1beta1.Testrun) (bool, error) {
		*r.Testrun = *new
		if r.Testrun.Status.State != "" {
			testrunPhase = r.Testrun.Status.Phase
			log.Info(fmt.Sprintf("Testrun %s is in %s phase. State: %s", r.Testrun.GetName(), testrunPhase, r.Testrun.Status.State))
		} else {
			log.Info(fmt.Sprintf("Testrun %s is in %s phase. Waiting ...", r.Testrun.GetName(), testrunPhase))
		}
		return util.Completed(testrunPhase), nil
	})
	if err != nil {
		r.Testrun.Status.Phase = tmv1beta1.PhaseStatusTimeout
		r.Error = trerrors.NewTimeoutError(fmt.Sprintf("maximum wait time of %d is exceeded by Testrun %s", config.Timeout, r.Testrun.GetName()))
	}
}

func GetArgoURL(tmClient client.Client, tr *tmv1beta1.Testrun) (string, error) {
	// get argo url from the argo ingress if possible
	// return err if the ingress cannot be found
	argoIngress := &v1beta1.Ingress{}
	err := tmClient.Get(context.TODO(), client.ObjectKey{Name: "argo-ui", Namespace: "default"}, argoIngress)
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

// GetClusterDomainURL tries to derive the cluster domain url from an grafana ingress if possible. Returns an error if the ingress cannot be found or is in unexpected form.
func GetClusterDomainURL(tmClient client.Client) (string, error) {
	// try to derive the cluster domain url from grafana ingress if possible
	// return err if the ingress cannot be found
	if tmClient == nil {
		return "", nil
	}
	ingress := &v1beta1.Ingress{}
	err := tmClient.Get(context.TODO(), client.ObjectKey{Namespace: "monitoring", Name: "grafana"}, ingress)
	if err != nil {
		return "", fmt.Errorf("cannot get grafana ingress: %v", err)
	}
	if len(ingress.Spec.Rules) == 0 {
		return "", fmt.Errorf("cannot get ingress rule from ingress %v", ingress)
	}
	host := ingress.Spec.Rules[0].Host
	r, _ := regexp.Compile("[a-z]+\\.ingress\\.(.+)$")
	matches := r.FindStringSubmatch(host)
	if len(matches) < 2 {
		return "", fmt.Errorf("cannot regex cluster domain from ingress %v", ingress)
	}
	return matches[1], nil
}
