package testrunner

import (
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

const (
	AnnotationLandscape     = "testrunner.testmachinery.sapcloud.io/landscape"
	AnnotationK8sVersion    = "testrunner.testmachinery.sapcloud.io/k8sVersion"
	AnnotationCloudProvider = "testrunner.testmachinery.sapcloud.io/cloudprovider"
)

func (m *Metadata) Annotations() map[string]string {
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
