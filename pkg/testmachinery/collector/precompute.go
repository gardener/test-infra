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

package collector

import (
	"fmt"
	"github.com/Masterminds/semver"
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
)

// PreComputeTeststepFields precomputes fields for elasticsearch that are otherwise hard to add at runtime (i.e. as grafana does not support scripted fields)
func PreComputeTeststepFields(phase argov1.NodePhase, k8sVersion string, clusterDomain string) *metadata.StepPreComputed {
	var preComputed metadata.StepPreComputed

	switch phase {
	case tmv1beta1.PhaseStatusFailed, tmv1beta1.PhaseStatusTimeout:
		zero := 0
		preComputed.PhaseNum = &zero
	case tmv1beta1.PhaseStatusSuccess:
		hundred := 100
		preComputed.PhaseNum = &hundred
	}

	if k8sVersion != "" {
		semVer, err := semver.NewVersion(k8sVersion)
		if err != nil {
			fmt.Printf("cannot parse k8s Version, will not precompute derived data: %s", err)
		} else {
			preComputed.K8SMajorMinorVersion = fmt.Sprintf("%d.%d", semVer.Major(), semVer.Minor())
		}
	}

	if clusterDomain != "" {
		preComputed.ArgoDisplayName = "argo"
		preComputed.LogsDisplayName = "logs"
		preComputed.ClusterDomain = clusterDomain
	}

	return &preComputed
}
