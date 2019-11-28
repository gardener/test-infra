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

package testrunner

import (
	"fmt"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

const (
	AnnotationLandscape       = "testrunner.testmachinery.sapcloud.io/landscape"
	AnnotationK8sVersion      = "testrunner.testmachinery.sapcloud.io/k8sVersion"
	AnnotationCloudProvider   = "testrunner.testmachinery.sapcloud.io/cloudprovider"
	AnnotationOperatingSystem = "testrunner.testmachinery.sapcloud.io/operating-system"
	AnnotationDimension       = "testrunner.testmachinery.sapcloud.io/dimension"
)

// CreateAnnotations creates annotations of the metadata to be set on the respective workflow
func (m *Metadata) CreateAnnotations() map[string]string {
	return map[string]string{
		AnnotationLandscape:       m.Landscape,
		AnnotationK8sVersion:      m.KubernetesVersion,
		AnnotationCloudProvider:   m.CloudProvider,
		AnnotationOperatingSystem: m.OperatingSystem,
		AnnotationDimension:       m.GetDimensionFromMetadata(),
	}
}

// GetDimensionFromMetadata returns a string describing the dimension of the metadata
func (m *Metadata) GetDimensionFromMetadata() string {
	d := fmt.Sprintf("%s/%s/%s", m.CloudProvider, m.KubernetesVersion, m.OperatingSystem)
	if m.FlavorDescription != "" {
		d = fmt.Sprintf("%s-%s", d, m.FlavorDescription)
	}
	return d
}

// DeepCopy creates a copy of the metadata struct
// todo: deep copy annotations and components if set
func (m *Metadata) DeepCopy() *Metadata {
	meta := *m
	return &meta
}

// MetadataFromTestrun reads metadata from a testrun
func MetadataFromTestrun(tr *tmv1beta1.Testrun) *Metadata {
	return &Metadata{
		Landscape:         tr.Annotations[AnnotationLandscape],
		KubernetesVersion: tr.Annotations[AnnotationK8sVersion],
		CloudProvider:     tr.Annotations[AnnotationCloudProvider],
		OperatingSystem:   tr.Annotations[AnnotationOperatingSystem],
		Testrun: TestrunMetadata{
			ID: tr.Name,
		},
	}
}
