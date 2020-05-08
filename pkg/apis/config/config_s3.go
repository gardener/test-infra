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

import "encoding/json"

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
