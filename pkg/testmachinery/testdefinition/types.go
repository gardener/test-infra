package testdefinition

import (
	"fmt"
	"path"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	apiv1 "k8s.io/api/core/v1"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/config"
)

// TestDefinition represents a TestDefinition which was fetched from locations.
type TestDefinition struct {
	Info     *tmv1beta1.TestDefinition
	Location Location
	FileName string
	Template *argov1.Template

	Volumes []apiv1.Volume

	inputArtifacts  ArtifactSet
	outputArtifacts ArtifactSet
	config          config.Set
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
	// GitInfo returns the current git information
	GitInfo() GitInfo
}

// GitInfo describes additional information about the used sources.
type GitInfo struct {
	SHA string
	Ref string
}

// GetStdInputArtifacts returns the default input artifacts of testdefionitions.
// The artifacts include kubeconfigs and shared folder inputs
func GetStdInputArtifacts() []argov1.Artifact {
	return []argov1.Artifact{
		{
			Name:     testmachinery.ArtifactKubeconfigs,
			Path:     testmachinery.TM_KUBECONFIG_PATH,
			Optional: true,
		},
		{
			Name:     testmachinery.ArtifactSharedFolder,
			Path:     testmachinery.TM_SHARED_PATH,
			Optional: true,
		},
	}
}

// GetUntrustedInputArtifacts returns the untrusted input artifacts of testdefionitions.
// The artifacts only include minimal configuration
func GetUntrustedInputArtifacts() []argov1.Artifact {
	return []argov1.Artifact{
		{
			Name:     testmachinery.ArtifactUntrustedKubeconfigs,
			Path:     path.Join(testmachinery.TM_KUBECONFIG_PATH, tmv1beta1.ShootKubeconfigName),
			Optional: true,
		},
	}
}

// GetStdOutputArtifacts returns the default output artifacts of a step.
// These artifacts include kubeconfigs and the shared folder.
func GetStdOutputArtifacts(global bool) []argov1.Artifact {
	kubeconfigArtifact := argov1.Artifact{
		Name:     testmachinery.ArtifactKubeconfigs,
		Path:     testmachinery.TM_KUBECONFIG_PATH,
		Optional: true,
	}
	untrustedKubeconfigArtifact := argov1.Artifact{
		Name:     testmachinery.ArtifactUntrustedKubeconfigs,
		Path:     path.Join(testmachinery.TM_KUBECONFIG_PATH, tmv1beta1.ShootKubeconfigName),
		Optional: true,
	}
	sharedFolderArtifact := argov1.Artifact{
		Name:     testmachinery.ArtifactSharedFolder,
		Path:     testmachinery.TM_SHARED_PATH,
		Optional: true,
	}

	if global {
		kubeconfigArtifact.GlobalName = kubeconfigArtifact.Name
		kubeconfigArtifact.Name = fmt.Sprintf("%s-global", kubeconfigArtifact.Name)
		untrustedKubeconfigArtifact.GlobalName = untrustedKubeconfigArtifact.Name
		untrustedKubeconfigArtifact.Name = fmt.Sprintf("%s-global", untrustedKubeconfigArtifact.Name)

		sharedFolderArtifact.GlobalName = sharedFolderArtifact.Name
		sharedFolderArtifact.Name = fmt.Sprintf("%s-global", sharedFolderArtifact.Name)
	}

	return []argov1.Artifact{kubeconfigArtifact, untrustedKubeconfigArtifact, sharedFolderArtifact}
}
