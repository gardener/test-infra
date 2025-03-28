// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"path/filepath"
)

// ChartsPath is the path to the charts
var ChartsPath = filepath.Join("charts", "internal")

// GitHubSecretKeyName is the name of the secret key that contains the github secrets
const GitHubSecretKeyName = "config.yaml" // #nosec G101 -- No credential.

// S3SecretName is the name of the secret containing the s3 credentials
const S3SecretName = "s3-secret"

// Argo constants
const (
	// ArgoChartName is the name of the chart to bootstrap argo
	ArgoChartName = "argo"

	// ArgoManagedResourceName is the name of the managed resource deployment
	ArgoManagedResourceName = "argo"

	// ArgoUIImageName is the name of the argo ui image in the image vector
	ArgoUIImageName = "argo-server"

	// ArgoWorkflowControllerImageName is the name of the argo workflow controller image in the image vector
	ArgoWorkflowControllerImageName = "argo-workflow-controller"

	// ArgoExecutorImageName is the name of the argo executor image in the image vector
	ArgoExecutorImageName = "argo-executor"

	// ArgoUIIngressName is the name of the argo ui ingress resource deployed to the cluster
	ArgoUIIngressName = "argo-server"

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

// Reserve excess capacity
const (
	// ReserveExcessCapacityChartName is the name of the chart to deploy reserved excess capacity pods
	ReserveExcessCapacityChartName = "reserve-excess-capacity"

	// ReserveExcessCapacityManagedResourceName is the name of the managed resource for reserved excess capicity pods
	ReserveExcessCapacityManagedResourceName = "reserve-excess-capacity"

	// ReserveExcessCapacityImageName is the name of the image for reserved excess capicity
	ReserveExcessCapacityImageName = "reserve-excess-capacity"
)
