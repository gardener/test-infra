package testdefinition

import (
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
	apiv1 "k8s.io/api/core/v1"
)

// TestDefinition represents a TestDefinition which was fetched from locations.
type TestDefinition struct {
	Info     *tmv1beta1.TestDefinition
	TaskName string
	Location Location
	FileName string
	Template *argov1.Template
	Config   []*config.Element

	Volumes []apiv1.Volume

	inputArtifacts  map[string]bool
	outputArtifacts map[string]bool
}

// Location is an interface for different testDefLocation types like git or local
type Location interface {
	// SetTestDefs adds Testdefinitions to the map.
	SetTestDefs(map[string]*TestDefinition) error
	// Type returns the tmv1beta1.LocationType type.
	Type() tmv1beta1.LocationType
	// Name returns the unique name of the location.
	Name() string
	// GetLocation returns the original TestLocation object
	GetLocation() *tmv1beta1.TestLocation
}

// GetStdInputArtifacts returns the default input artifacts of testdefionitions.
// Thes artifacts include kubeconfig and shared folder inputs
func GetStdInputArtifacts() []argov1.Artifact {
	return []argov1.Artifact{
		{
			Name:     "kubeconfigs",
			Path:     testmachinery.TM_KUBECONFIG_PATH,
			Optional: true,
		},
		{
			Name:     "sharedFolder",
			Path:     testmachinery.TM_SHARED_PATH,
			Optional: true,
		},
	}
}
