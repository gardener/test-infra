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

package v1beta1

import "encoding/json"

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
