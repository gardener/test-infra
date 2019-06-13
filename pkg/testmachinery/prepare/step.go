package prepare

import "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"

func GetPrepareStep(useGlobalArtifacts bool) *v1beta1.DAGStep {
	return &v1beta1.DAGStep{
		Name:               "prepare",
		UseGlobalArtifacts: useGlobalArtifacts,
	}
}
