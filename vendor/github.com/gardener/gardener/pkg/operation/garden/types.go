// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package garden

import (
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
)

// Garden is an object containing Garden cluster specific data.
type Garden struct {
	Project        *gardenv1beta1.Project
	DefaultDomains []*DefaultDomain
	InternalDomain *InternalDomain
}

// InternalDomain contains information about the internal domain configured in the garden cluster.
type InternalDomain struct {
	Domain     string
	Provider   string
	SecretData map[string][]byte
}

// DefaultDomain contains information about a default domain configured in the garden cluster.
type DefaultDomain struct {
	Domain     string
	Provider   string
	SecretData map[string][]byte
}
