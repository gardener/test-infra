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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Configuration contains the testmachinery configuration values
type Configuration struct {
	metav1.TypeMeta        `json:",inline"`
	Controller             Controller              `json:"controller"`
	TestMachinery          TestMachinery           `json:"testmachinery"`
	Argo                   Argo                    `json:"argo"`
	GitHub                 GitHub                  `json:"github,omitempty"`
	S3                     *S3                     `json:"s3Configuration,omitempty"`
	ElasticSearch          *ElasticSearch          `json:"esConfiguration,omitempty"`
	ReservedExcessCapacity *ReservedExcessCapacity `json:"reservedExcessCapacity,omitempty"`
	Observability          Observability           `json:"observability,omitempty"`
}

// Controller holds information about the testmachinery controller
type Controller struct {
	// HealthAddr is the address of the healtcheck endpoint.
	HealthAddr string `json:"healthAddr,omitempty"`

	// MetricsAddr is the address of the metrics endpoint.
	MetricsAddr string `json:"metricsAddr,omitempty"`

	// EnableLeaderElection enables leader election for the controller.
	EnableLeaderElection bool `json:"enableLeaderElection,omitempty"`

	// MaxConcurrentSyncs is the max concurrent reconciles the controller does.
	MaxConcurrentSyncs int `json:"maxConcurrentSyncs,omitempty"`

	// TTLController contains the ttl controller configuration.
	TTLController TTLController `json:"ttlController,omitempty"`

	// WebhookConfig holds the validating webhook configuration.
	WebhookConfig WebhookConfig `json:"webhook,omitempty"`
}

// TTLController contains the ttl controller configuration.
type TTLController struct {
	// Disable disables the ttl controller.
	Disable bool `json:"disable,omitempty"`
	// MaxConcurrentSyncs is the max concurrent reconciles the controller does.
	MaxConcurrentSyncs int `json:"maxConcurrentSyncs,omitempty"`
}

// WebhookConfig holds the validating webhook configuration
type WebhookConfig struct {
	// Port is the port to serve validating webhooks
	Port int `json:"port,omitempty"`

	// CertDir is the directory that contains the certificates that is used by the webhook
	CertDir string `json:"certDir,omitempty"`
}

// ElasticSearch holds information about the elastic instance to write data to.
type ElasticSearch struct {
	Endpoint string `json:"endpoint,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ReservedExcessCapacity holds information about additionally deployed reserved excess capacity pods.
type ReservedExcessCapacity struct {
	// Replicas is the amount of reserve excess capacity pods.
	Replicas int32 `json:"replicas"`

	// Resources specifies the resources of the single excess capacity pods
	// + optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// Ingress holds information about a ingress
type Ingress struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
}
