// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
)

// PreComputeTeststepFields precomputes fields for elasticsearch that are otherwise hard to add at runtime (i.e. as grafana does not support scripted fields)
func PreComputeTeststepFields(phase argov1.NodePhase, meta metadata.Metadata, clusterDomain string) *metadata.StepPreComputed {
	var preComputed metadata.StepPreComputed

	switch phase {
	case tmv1beta1.StepPhaseFailed, tmv1beta1.StepPhaseTimeout:
		zero := 0
		preComputed.PhaseNum = &zero
	case tmv1beta1.StepPhaseSuccess:
		hundred := 100
		preComputed.PhaseNum = &hundred
	}

	if meta.KubernetesVersion != "" {
		semVer, err := semver.NewVersion(meta.KubernetesVersion)
		if err != nil {
			fmt.Printf("cannot parse k8s Version '%s', will try to strip double quotes: %s\n", meta.KubernetesVersion, err)
			semVer, err = semver.NewVersion(strings.Replace(meta.KubernetesVersion, "\"", "", -1))
			if err != nil {
				fmt.Printf("still cannot parse k8s Version, cannot precompute k8sMajMin Version: %s", err)
			}
		}
		if err == nil {
			preComputed.K8SMajorMinorVersion = fmt.Sprintf("%d.%d", semVer.Major(), semVer.Minor())
		}
	}

	if clusterDomain != "" {
		preComputed.ArgoDisplayName = "argo"
		preComputed.LogsDisplayName = "logs"
		preComputed.ClusterDomain = clusterDomain
	}

	providerEnhanced := meta.CloudProvider
	if meta.CloudProvider == "openstack" && meta.Region != "" {
		providerEnhanced += "_" + meta.Region
	}
	if meta.FlavorDescription != "" {
		providerEnhanced += "_" + meta.FlavorDescription
	}
	if meta.AllowPrivilegedContainers != nil && !*meta.AllowPrivilegedContainers {
		providerEnhanced += "(NoPrivCtrs)"
	}
	if meta.ContainerRuntime != "" {
		providerEnhanced += fmt.Sprintf("{%s}", meta.ContainerRuntime)
	}
	preComputed.ProviderEnhanced = providerEnhanced

	return &preComputed
}
