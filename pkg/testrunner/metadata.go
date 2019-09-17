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
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

const (
	AnnotationLandscape     = "testrunner.testmachinery.sapcloud.io/landscape"
	AnnotationK8sVersion    = "testrunner.testmachinery.sapcloud.io/k8sVersion"
	AnnotationCloudProvider = "testrunner.testmachinery.sapcloud.io/cloudprovider"
)

func (m *Metadata) CreateAnnotations() map[string]string {
	return map[string]string{
		AnnotationLandscape:     m.Landscape,
		AnnotationK8sVersion:    m.KubernetesVersion,
		AnnotationCloudProvider: m.CloudProvider,
	}
}

func MetadataFromTestrun(tr *tmv1beta1.Testrun) *Metadata {
	return &Metadata{
		Landscape:         tr.Annotations[AnnotationLandscape],
		KubernetesVersion: tr.Annotations[AnnotationK8sVersion],
		CloudProvider:     tr.Annotations[AnnotationCloudProvider],
		Testrun: TestrunMetadata{
			ID: tr.Name,
		},
	}
}
