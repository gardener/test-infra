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

package config

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration contains the testmachinery configuration values
type Configuration struct {
	metav1.TypeMeta `json:",inline"`
	Controller      Controller     `json:"controller"`
	TestMachinery   TestMachinery  `json:"testmachinery"`
	GitHub          GitHub         `json:"github,omitempty"`
	S3              *S3            `json:"s3Configuration,omitempty"`
	ElasticSearch   *ElasticSearch `json:"esConfiguration,omitempty"`
	Argo            Argo           `json:"argo"`
	Observability   Observability  `json:"observability,omitempty"`
}

// Controller holds information about the testmachinery controller
type Controller struct {
	// HealthAddr is the address of the healtcheck endpoint
	HealthAddr string `json:"healthAddr,omitempty"`

	// MetricsAddr is the address of the metrics endpoint
	MetricsAddr string `json:"metricsAddr,omitempty"`

	// EnableLeaderElection enables leader election for the controller
	EnableLeaderElection bool `json:"enableLeaderElection,omitempty"`

	// MaxConcurrentSyncs is the max concurrent reconciles the controller does.
	MaxConcurrentSyncs int `json:"maxConcurrentSyncs,omitempty"`

	// WebhookConfig holds the validating webhook configuration
	WebhookConfig WebhookConfig `json:"webhook,omitempty"`
}

// WebhookConfig holds the validating webhook configuration
type WebhookConfig struct {
	Port    int    `json:"port,omitempty"`
	CertDir string `json:"certDir,omitempty"`
}

// TestMachinery holds information about the testmachinery
type TestMachinery struct {
	// Namespace is the namespace the testmachinery is deployed to.
	Namespace string `json:"namespace,omitempty"`

	// TestDefPath is the repository path where the Test Machinery should search for testdefinitions.
	TestDefPath string `json:"testdefPath"`

	// PrepareImage is the prepare image that is used in the prepare and postprepare step.
	PrepareImage string `json:"prepareImage"`

	// PrepareImage is the base image that is used as the default image if a TestDefinition does not define an image.
	BaseImage string `json:"baseImage"`

	// Local indicates if the controller is run locally.
	Local bool `json:"local,omitempty"`

	// Insecure indicates that the testmachinery runs insecure.
	Insecure bool `json:"insecure,omitempty"`

	// DisableCollector disables the collection of test results and their ingestion into elasticsearch.
	DisableCollector bool `json:"disableCollector"`

	// CleanWorkflowPods indicates if workflow pods should be directly cleaned up by the testmachinery.
	CleanWorkflowPods bool `json:"cleanWorkflowPods,omitempty"`
}

// GitHub holds all github related information needed in the testmachinery.
type GitHub struct {
	Cache *GitHubCache `json:"cache,omitempty"`

	// SecretsPath is the path to the github secrets file
	SecretsPath string `json:"secretsPath,omitempty"`
}

// GitHubCache is the github cache configuration
type GitHubCache struct {
	CacheDir        string `json:"cacheDir,omitempty"`
	CacheDiskSizeGB int    `json:"cacheDiskSizeGB,omitempty"`
	MaxAgeSeconds   int    `json:"maxAgeSeconds,omitempty"`
}

// Argo holds configuration for the argo installation
type Argo struct {
	// Ingress holds the argo ui ingress configuration
	ArgoUI ArgoUI `json:"argoUI"`

	// Specify additional values that are passed to the argo helm chart
	// +optional
	ChartValues json.RawMessage `json:"chartValues,omitempty"`
}

// ArgoUI holds information about the argo ui to deploy
type ArgoUI struct {
	// Ingress holds the argo ui ingress configuration
	Ingress Ingress `json:"ingress"`
}

// S3 holds information about the s3 endpoint
type S3 struct {
	Server     S3Server `json:"server"`
	BucketName string   `json:"bucketName,omitempty"`
	AccessKey  string   `json:"accessKey,omitempty"`
	SecretKey  string   `json:"secretKey,omitempty"`
}

// S3Server defines the used s3 server
// The endpoint and ssl is not needed if minio should be deployed.
// Minio is deployed when the struct is defined
type S3Server struct {
	// +optional
	Minio *MinioConfiguration `json:"minio"`

	Endpoint string `json:"endpoint,omitempty"`
	SSL      bool   `json:"ssl,omitempty"`
}

// MinioConfiguration configures optional minio deployment
type MinioConfiguration struct {
	// Distributed specified that minio should be deployed in cluster mode
	Distributed bool `json:"distributed"`

	// Ingress is the ingress configuration to expose minio
	// +optional
	Ingress Ingress `json:"ingress,omitempty"`

	// Specify additional values that are passed to the minio helm chart
	// +optional
	ChartValues json.RawMessage `json:"chartValues,omitempty"`
}

// ElasticSearch holds information about the elastic instance to write data to.
type ElasticSearch struct {
	Endpoint string `json:"endpoint,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// Observability holds the configuration for logging and monitoring tooling
type Observability struct {
	// Logging configures the logging stack
	// will not be deployed if empty
	Logging *Logging `json:"logging,omitempty"`
}

// Logging holds the configuration for the loki/promtail logging stack
type Logging struct {
	// Namespace configures the namespace the logging stack is deployed to.
	Namespace string `json:"namespace"`

	// StorageClass configures the storage class for the loki deployment
	StorageClass string `json:"storageClass"`

	// Specify additional values that are passed to the minio helm chart
	// +optional
	ChartValues json.RawMessage `json:"chartValues,omitempty"`
}

// Ingress holds information about a ingress
type Ingress struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
}
