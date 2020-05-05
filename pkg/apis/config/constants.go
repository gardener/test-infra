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

package config

import "path/filepath"

// ChartsPath is the path to the charts
var ChartsPath = filepath.Join("charts", "internal")

// GitHubSecretKeyName is the name of the secret key that contains the github secrets
const GitHubSecretKeyName = "config.yaml"

// S3SecretName is the name of the secret containing the s3 credentials
const S3SecretName = "s3-secret"

// ResourceManagerDeploymentName is the name of the gardener resource manager deployment
const ResourceManagerDeploymentName = "gardener-resource-manager"

// Argo constants
const (
	// ArgoChartName is the name of the chart to bootstrap argo
	ArgoChartName = "argo"

	// ArgoManagedResourceName is the name of the managed resource deployment
	ArgoManagedResourceName = "argo"

	// ArgoUIImageName is the name of the argo ui image in the image vector
	ArgoUIImageName = "argo-ui"

	// ArgoWorkflowControllerImageName is the name of the argo workflow controller image in the image vector
	ArgoWorkflowControllerImageName = "argo-workflow-controller"

	// ArgoExecutorImageName is the name of the argo executor image in the image vector
	ArgoExecutorImageName = "argo-executor"

	// ArgoUIIngressName is the name of the argo ui ingress resource deployed to the cluster
	ArgoUIIngressName = "argo-ui"

	// ArgoWorkflowControllerDeploymentName is the name workflow controller deployment
	ArgoWorkflowControllerDeploymentName = "workflow-controller"
)

// Minio constants
const (
	// MinioChartName is the name of the chart to bootstrap minio
	MinioChartName = "minio"

	// MinioManagedResourceName is the name of the managed resource deployment
	MinioManagedResourceName = "minio"

	// MinioImageName is the name of the minio image in the image vector
	MinioImageName = "minio"

	// MinioDeploymentName is the name of the minio deployment or statefulset in the cluster
	MinioDeploymentName = "minio"

	// MinioServiceName is the name of the minio service in the cluster
	MinioServiceName = "minio"

	// MinioServicePort is the port of the minio service in the cluster
	MinioServicePort = 9000
)

// Logging constants
const (
	// LoggingChartName is the name of the chart to bootstrap logging
	LoggingChartName = "logging"

	// LoggingManagedResourceName is the name of the managed resource deployment
	LoggingManagedResourceName = "logging"

	// LokiImageName is the name of the loki image in the image vector
	LokiImageName = "loki"

	// PromtailImageName is the name of the promtail image in the image vector
	PromtailImageName = "promtail"
)
