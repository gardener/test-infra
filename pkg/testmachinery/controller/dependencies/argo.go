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

package dependencies

import (
	"context"
	"encoding/json"
	"fmt"
	argov1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/utils"
	"github.com/gardener/gardener/pkg/utils/chart"
	intconfig "github.com/gardener/test-infra/pkg/apis/config"
	"github.com/gardener/test-infra/pkg/testmachinery/imagevector"
	"github.com/gardener/test-infra/pkg/testrunner"
)

func (e *DependencyEnsurer) ensureArgo(ctx context.Context, namespace string, config *intconfig.Configuration) error {
	e.log.Info("Ensuring argo deployment")
	values := map[string]interface{}{
		"argo": map[string]interface{}{
			"name": intconfig.ArgoWorkflowControllerDeploymentName,
			"logging": map[string]interface{}{
				"links": e.getExternalLoggingLinks(ctx),
			},
		},
		"argoui": map[string]interface{}{
			"ingress": map[string]interface{}{
				"enabled": config.Argo.ArgoUI.Ingress.Enabled,
				"name":    intconfig.ArgoUIIngressName,
				"host":    config.Argo.ArgoUI.Ingress.Host,
			},
		},
		"objectStorage": map[string]interface{}{
			"bucketName": config.S3.BucketName,
			"endpoint":   config.S3.Server.Endpoint,
			"secret": map[string]string{
				"name": intconfig.S3SecretName,
			},
		},
	}

	if config.Argo.ChartValues != nil {
		additionalValues := map[string]interface{}{}
		if err := json.Unmarshal(config.Argo.ChartValues, &additionalValues); err != nil {
			return err
		}

		values = utils.MergeMaps(additionalValues, values)
	}

	values, err := chart.InjectImages(values, imagevector.ImageVector(), []string{
		intconfig.ArgoUIImageName,
		intconfig.ArgoWorkflowControllerImageName,
		intconfig.ArgoExecutorImageName,
	})
	if err != nil {
		return fmt.Errorf("failed to find image version %v", err)
	}

	err = e.createManagedResource(ctx, namespace, intconfig.ArgoManagedResourceName, e.renderer,
		intconfig.ArgoChartName, values, nil)
	if err != nil {
		e.log.Error(err, "unable to create managed resource")
		return err
	}
	return nil
}

// getExternalLoggingLinks returns argo links for the grafana logging if it is deployed
func (e *DependencyEnsurer) getExternalLoggingLinks(ctx context.Context) []argov1alpha1.Link {
	links := make([]argov1alpha1.Link, 0)

	grafanaHost, err := testrunner.GetGrafanaHost(ctx, e.client)
	if err != nil {
		e.log.Error(err, "unable to get grafana host")
		return nil
	}

	links = append(links, argov1alpha1.Link{
		Name:  "Grafana Workflow Log",
		Scope: "workflow",
		URL:   testrunner.GetGrafanaURLFromHostForWorkflow(grafanaHost, "${metadata.name}"),
	},
		argov1alpha1.Link{
			Name:  "Grafana Pod Log",
			Scope: "pod",
			URL:   testrunner.GetGrafanaURLFromHostForPod(grafanaHost, "${metadata.name}"),
		})

	return links
}
