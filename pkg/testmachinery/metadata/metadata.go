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

package metadata

import (
	"fmt"
	"strconv"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/util"
)

// CreateAnnotations creates annotations of the metadata to be set on the respective workflow
func (m *Metadata) CreateAnnotations() map[string]string {
	annotations := map[string]string{
		common.AnnotationLandscape:              m.Landscape,
		common.AnnotationK8sVersion:             m.KubernetesVersion,
		common.AnnotationCloudProvider:          m.CloudProvider,
		common.AnnotationOperatingSystem:        m.OperatingSystem,
		common.AnnotationOperatingSystemVersion: m.OperatingSystemVersion,
		common.AnnotationContainerRuntime:       m.ContainerRuntime,
		common.AnnotationRegion:                 m.Region,
		common.AnnotationZone:                   m.Zone,
		common.AnnotationFlavorDescription:      m.FlavorDescription,
		common.AnnotationDimension:              m.GetDimensionFromMetadata("/"),
		common.AnnotationRetries:                strconv.Itoa(m.Retries),
		common.AnnotationShootAnnotations:       util.MarshalMap(m.Annotations),
	}
	if m.AllowPrivilegedContainers != nil {
		annotations[common.AnnotationAllowPrivilegedContainers] = strconv.FormatBool(*m.AllowPrivilegedContainers)
	}
	return annotations
}

// GetDimensionFromMetadata returns a string describing the dimension of the metadata
func (m *Metadata) GetDimensionFromMetadata(sep string) string {
	d := fmt.Sprintf("%s"+sep+"%s"+sep+"%s", m.CloudProvider, m.KubernetesVersion, m.OperatingSystem)
	if m.AllowPrivilegedContainers != nil && !*m.AllowPrivilegedContainers {
		d = fmt.Sprintf("%s"+sep+"%s", d, "NoPrivCtrs")
	}
	if m.FlavorDescription != "" {
		d = fmt.Sprintf("%s"+sep+"%s", d, m.FlavorDescription)
	}
	return d
}

// DeepCopy creates a copy of the metadata struct
// todo: deep copy annotations and components if set
func (m *Metadata) DeepCopy() *Metadata {
	meta := *m
	return &meta
}

// FromTestrun reads metadata from a testrun
func FromTestrun(tr *tmv1beta1.Testrun) *Metadata {
	retries, _ := strconv.Atoi(tr.Annotations[common.AnnotationRetries])
	shootAnnotations, err := util.UnmarshalMap(tr.Annotations[common.AnnotationShootAnnotations])
	if err != nil {
		shootAnnotations = make(map[string]string)
		shootAnnotations["error"] = err.Error()
	}
	metadata := &Metadata{
		Landscape:              tr.Annotations[common.AnnotationLandscape],
		KubernetesVersion:      tr.Annotations[common.AnnotationK8sVersion],
		CloudProvider:          tr.Annotations[common.AnnotationCloudProvider],
		OperatingSystem:        tr.Annotations[common.AnnotationOperatingSystem],
		OperatingSystemVersion: tr.Annotations[common.AnnotationOperatingSystemVersion],
		ContainerRuntime:       tr.Annotations[common.AnnotationContainerRuntime],
		Region:                 tr.Annotations[common.AnnotationRegion],
		Zone:                   tr.Annotations[common.AnnotationZone],
		FlavorDescription:      tr.Annotations[common.AnnotationFlavorDescription],
		ShootAnnotations:       shootAnnotations,
		Retries:                retries,
		Testrun: TestrunMetadata{
			ID:             tr.Name,
			StartTime:      tr.Status.StartTime,
			ExecutionGroup: tr.Labels[common.LabelTestrunExecutionGroup],
		},
	}
	if a, ok := tr.Annotations[common.AnnotationAllowPrivilegedContainers]; ok {
		if b, err := strconv.ParseBool(a); err == nil {
			metadata.AllowPrivilegedContainers = &b
		}
	}
	return metadata
}
