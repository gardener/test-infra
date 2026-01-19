// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testrunner

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/pkg/errors"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/testmachinery"
)

func GetArgoURL(ctx context.Context, k8sClient client.Client, tr *tmv1beta1.Testrun) (string, error) {
	argoBaseURL, err := GetArgoHost(ctx, k8sClient)
	if err != nil {
		return "", err
	}
	return GetArgoURLFromHost(argoBaseURL, tr), nil
}

// GetArgoURLFromHost returns the url for a specific workflow with a given base path
func GetArgoURLFromHost(host string, tr *tmv1beta1.Testrun) string {
	return fmt.Sprintf("%s/%s", host, path.Join("workflows", tr.Namespace, testmachinery.GetWorkflowName(tr)))
}

// GetArgoHost returns the host of the argo ui
func GetArgoHost(ctx context.Context, tmClient client.Client) (string, error) {
	return GetHostURLFromIngress(ctx, tmClient, client.ObjectKey{Name: "argo-server", Namespace: "default"})
}

// GetGrafanaURLFromHostForWorkflow returns the path to the logs in grafana for a whole workflow
func GetGrafanaURLFromHostForWorkflow(host string, workflowName string) string {
	return fmt.Sprintf(`%s/explore?left=["now-3d","now","Vali",{"expr":"{container%%3D\"main\",argo_workflow%%3D\"%s\"}"},{"mode":"Logs"},{"ui":[true,true,true,"exact"]}]`, host, workflowName)
}

// GetGrafanaURLFromHostForStep returns the path to the logs in grafana for a specific step
func GetGrafanaURLFromHostForStep(host string, workflowName, testdefName string) string {
	return fmt.Sprintf(`%s/explore?left=["now-3d","now","Vali",{"expr":"{container%%3D\"main\",tm_testdef%%3D\"%s\",argo_workflow%%3D\"%s\"}"},{"mode":"Logs"},{"ui":[true,true,true,"exact"]}]`, host, testdefName, workflowName)
}

// GetGrafanaURLFromHostForPod returns the path to the logs in grafana for a specific pod
func GetGrafanaURLFromHostForPod(host string, podname string) string {
	return fmt.Sprintf(`%s/explore?left=["now-3d","now","Vali",{"expr":"{container%%3D\"main\",instance%%3D\"%s\"}"},{"mode":"Logs"},{"ui":[true,true,true,"exact"]}]`, host, podname)
}

// GetGrafanaHost returns the host of the grafana instance in the monitoring namespace
func GetGrafanaHost(ctx context.Context, tmClient client.Client) (string, error) {
	return GetHostURLFromIngress(ctx, tmClient, client.ObjectKey{Namespace: "monitoring", Name: "plutono"})
}

// GetTmDashboardURLForTestrun returns the dashboard URL to a testrun
func GetTmDashboardURLForTestrun(tmClient client.Client, tr *tmv1beta1.Testrun) (string, error) {
	host, err := GetTMDashboardHost(tmClient)
	if err != nil {
		return "", err
	}
	return GetTmDashboardURLFromHostForTestrun(host, tr), nil
}

// GetTmDashboardURLFromHostForExecutionGroup returns the dashboard URL to a execution group with a given dashboard host
func GetTmDashboardURLFromHostForExecutionGroup(host, executiongroupID string) string {
	return fmt.Sprintf("%s/testruns?%s=%s", host, common.DashboardExecutionGroupParameter, executiongroupID)
}

// GetTmDashboardURLFromHostForTestrun returns the dashboard URL to a testrun with a given dashboard host
func GetTmDashboardURLFromHostForTestrun(host string, tr *tmv1beta1.Testrun) string {
	return fmt.Sprintf("%s/testrun/%s/%s", host, tr.Namespace, tr.Name)
}

// GetTMDashboardHost returns the host of the TestMachinery Dashboard
func GetTMDashboardHost(tmClient client.Client) (string, error) {
	ingressList := &netv1.IngressList{}
	if err := tmClient.List(context.TODO(), ingressList, client.MatchingLabels{common.LabelTMDashboardIngress: "true"}); err != nil {
		return "", errors.Wrapf(err, "unable to list TestMachinery Dashboard ingress with label %s", common.LabelTMDashboardIngress)
	}

	if len(ingressList.Items) == 0 {
		return "", errors.Errorf("no ingresses found for TestMachinery Dashboard with label %s", common.LabelTMDashboardIngress)
	}

	return GetHostURLFromIngressObject(&ingressList.Items[0])
}

// GetClusterDomainURL tries to derive the cluster domain url from an grafana ingress if possible. Returns an error if the ingress cannot be found or is in unexpected form.
func GetHostURLFromIngress(ctx context.Context, tmClient client.Client, obj types.NamespacedName) (string, error) {
	// try to derive the cluster domain url from grafana ingress if possible
	// return err if the ingress cannot be found
	if tmClient == nil {
		return "", nil
	}
	ingress := &netv1.Ingress{}
	err := tmClient.Get(ctx, obj, ingress)
	if err != nil {
		return "", errors.Errorf("cannot get grafana ingress: %v", err)
	}

	return GetHostURLFromIngressObject(ingress)
}

// GetHostURLFromIngressObject tries to derive the cluster domain url from an ingeress object. Returns an error if the ingress is in unexpected form.
func GetHostURLFromIngressObject(ingress *netv1.Ingress) (string, error) {
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

// GetCloudProfilesFromDisk walks through the filesystem and returns all found CloudProfiles
func GetCloudProfilesFromDisk(cloudProfileSearchPath string) (map[string]gardencorev1beta1.CloudProfile, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(gardencorev1beta1.AddToScheme(scheme))
	decoder := serializer.NewCodecFactory(scheme).UniversalDecoder()
	cloudProfiles := make(map[string]gardencorev1beta1.CloudProfile)
	cloudProfileSearchPath = filepath.Clean(cloudProfileSearchPath)

	if _, err := os.Stat(cloudProfileSearchPath); os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "CloudProfile search path %s does not exist", cloudProfileSearchPath)
	}
	err := filepath.WalkDir(cloudProfileSearchPath, func(path string, d os.DirEntry, err error) error {
		// fail fast on walk error and continue, if it's a directory
		if err != nil || d.IsDir() {
			return err
		}
		// process files only
		path = filepath.Clean(path)
		if !strings.HasPrefix(path, cloudProfileSearchPath) {
			return errors.New("invalid CloudProfile path")
		}
		rawFile, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		cpfl := &gardencorev1beta1.CloudProfile{}
		_, _, decodeErr := decoder.Decode(rawFile, nil, cpfl)
		// deliberately ignore decode errors as not all files are CloudProfiles
		// filter out partial cloudprofile files that have no .metadata.name
		if decodeErr != nil || cpfl.Name == "" {
			return err
		}
		cloudProfiles[cpfl.Name] = *cpfl
		return err
	})

	return cloudProfiles, err
}
