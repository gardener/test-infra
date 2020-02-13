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
	"fmt"
	"github.com/gardener/test-infra/pkg/common"
	"k8s.io/apimachinery/pkg/types"
	"net/url"
	"path"
	"regexp"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/pkg/errors"
	"k8s.io/api/extensions/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetArgoURL(k8sClient client.Client, tr *tmv1beta1.Testrun) (string, error) {
	argoBaseURL, err := GetArgoHost(k8sClient)
	if err != nil {
		return "", nil
	}
	return GetArgoURLFromHost(argoBaseURL, tr), nil
}

// GetArgoURLFromHost returns the url for a specific workflow with a given base path
func GetArgoURLFromHost(host string, tr *tmv1beta1.Testrun) string {
	return fmt.Sprintf("%s/%s", host, path.Join("workflows", tr.Namespace, testmachinery.GetWorkflowName(tr)))
}

// GetArgoHost returns the host of the argo ui
func GetArgoHost(tmClient client.Client) (string, error) {
	return GetHostURLFromIngress(tmClient, client.ObjectKey{Name: "argo-ui", Namespace: "default"})
}

// GetGrafanaURLFromHostForWorkflow returns the path to the logs in grafana for a whole workflow
func GetGrafanaURLFromHostForWorkflow(host string, tr *tmv1beta1.Testrun) string {
	return fmt.Sprintf(`%s/explore?left=["now-7d","now","Loki",{"expr":"{container%%3D\"main\",argo_workflow%%3D\"%s\"}"},{"mode":"Logs"},{"ui":[true,true,true,"exact"]}]`, host, tr.Status.Workflow)
}

// GetGrafanaURLFromHostForStep returns the path to the logs in grafana for a specific step
func GetGrafanaURLFromHostForStep(host string, tr *tmv1beta1.Testrun, step *tmv1beta1.StepStatus) string {
	return fmt.Sprintf(`%s/explore?left=["now-7d","now","Loki",{"expr":"{container%%3D\"main\",tm_testdef%%3D\"%s\",argo_workflow%%3D\"%s\"}"},{"mode":"Logs"},{"ui":[true,true,true,"exact"]}]`, host, step.TestDefinition.Name, tr.Status.Workflow)
}

// GetGrafanaHost returns the host of the grafana instance in the monitoring namespace
func GetGrafanaHost(tmClient client.Client) (string, error) {
	return GetHostURLFromIngress(tmClient, client.ObjectKey{Namespace: "monitoring", Name: "grafana"})
}

// GetTmDashboardURLForTestrun returns the dashboard URL to a testrun
func GetTmDashboardURLForTestrun(tmClient client.Client, tr *tmv1beta1.Testrun) (string, error) {
	host, err := GetTMDashboardHost(tmClient)
	if err != nil {
		return "", nil
	}
	return GetTmDashboardURLFromHostForTestrun(host, tr), nil
}

// GetTmDashboardURLFromHostForExecutionGroup returns the dashboard URL to a execution group with a given dashboard host
func GetTmDashboardURLFromHostForExecutionGroup(host, executiongroupID string) string {
	return fmt.Sprintf("%s/testruns?%s=%s", host, common.DashboardExecutionGroupParameter, executiongroupID)
}

// GetTmDashboardURLFromHostForTestrun returns the dashboard URL to a testrun with a given dashboard host
func GetTmDashboardURLFromHostForTestrun(host string, tr *tmv1beta1.Testrun) string {
	return fmt.Sprintf("%s/testruns/%s/%s", host, tr.Namespace, tr.Name)
}

// GetTMDashboardHost returns the host of the TestMachinery Dashboard
func GetTMDashboardHost(tmClient client.Client) (string, error) {
	ingressList := &v1beta1.IngressList{}
	if err := tmClient.List(context.TODO(), ingressList); err != nil {
		return "", errors.Wrapf(err, "unable to list TestMachinery Dashboard ingress with label %s", common.LabelTMDashboardIngress)
	}

	if len(ingressList.Items) == 0 {
		return "", errors.Errorf("no ingresses found for TestMachinery Dashboard with label %s", common.LabelTMDashboardIngress)
	}

	return GetHostURLFromIngressObject(&ingressList.Items[0])
}

// GetClusterDomainURL tries to derive the cluster domain url from a grafana ingress if possible. Returns an error if the ingress cannot be found or is in unexpected form.
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

// GetClusterDomainURL tries to derive the cluster domain url from an grafana ingress if possible. Returns an error if the ingress cannot be found or is in unexpected form.
func GetHostURLFromIngress(tmClient client.Client, obj types.NamespacedName) (string, error) {
	// try to derive the cluster domain url from grafana ingress if possible
	// return err if the ingress cannot be found
	if tmClient == nil {
		return "", nil
	}
	ingress := &v1beta1.Ingress{}
	err := tmClient.Get(context.TODO(), obj, ingress)
	if err != nil {
		return "", errors.Errorf("cannot get grafana ingress: %v", err)
	}

	return GetHostURLFromIngressObject(ingress)
}

// GetHostURLFromIngressObject tries to derive the cluster domain url from an ingeress object. Returns an error if the ingress is in unexpected form.
func GetHostURLFromIngressObject(ingress *v1beta1.Ingress) (string, error) {
	if len(ingress.Spec.Rules) == 0 {
		return "", errors.New("no rules defined")
	}
	rule := ingress.Spec.Rules[0]
	if len(rule.HTTP.Paths) == 0 {
		return "", errors.New("no http backend defined")
	}

	protocol := "http://"
	if len(ingress.Spec.TLS) != 0 {
		protocol = "https://"
	}
	hostUrl, err := url.ParseRequestURI(protocol + rule.Host)
	if err != nil {
		return "", err
	}
	return hostUrl.String(), nil
}
