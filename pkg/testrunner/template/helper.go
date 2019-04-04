package template

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/utils"
	"sort"

	"github.com/Masterminds/semver"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getK8sVersions returns all K8s version that should be rendered by the chart
func getK8sVersions(parameters *TestrunParameters) ([]string, error) {
	if !parameters.MakeVersionMatrix && parameters.K8sVersion != "" {
		return []string{parameters.K8sVersion}, nil
	}
	// if the kubernetes version is not set, get the latest version defined by the cloudprofile
	if !parameters.MakeVersionMatrix && parameters.K8sVersion == "" {
		version, err := getLatestK8sVersion(parameters.GardenKubeconfigPath, parameters.Cloudprofile, parameters.Cloudprovider)
		if err != nil {
			return nil, fmt.Errorf("'k8s-version' is not defined nor can it be read from the cloudprofile: %s", err.Error())
		}
		return []string{version}, nil
	}
	if parameters.MakeVersionMatrix {
		return getK8sVersionsFromCloudprofile(parameters.GardenKubeconfigPath, parameters.Cloudprofile, parameters.Cloudprovider)
	}

	return nil, fmt.Errorf("No K8s version can be specified")
}

func getK8sVersionsFromCloudprofile(gardenerKubeconfigPath, cloudprofile, cloudprovider string) ([]string, error) {
	ctx := context.Background()
	defer ctx.Done()
	k8sGardenClient, err := kubernetes.NewClientFromFile("", gardenerKubeconfigPath, client.Options{
		Scheme: kubernetes.GardenScheme,
	})
	if err != nil {
		return nil, err
	}

	profile := &gardenv1beta1.CloudProfile{}
	err = k8sGardenClient.Client().Get(ctx, types.NamespacedName{Name: cloudprofile}, profile)
	if err != nil {
		return nil, err
	}

	return getCloudproviderVersions(profile, cloudprovider)
}

func getLatestK8sVersion(gardenerKubeconfigPath, cloudprofile, cloudprovider string) (string, error) {
	rawVersions, err := getK8sVersionsFromCloudprofile(gardenerKubeconfigPath, cloudprofile, cloudprovider)
	if err != nil {
		return "", err
	}

	if len(rawVersions) == 0 {
		return "", fmt.Errorf("No kubernetes versions found for cloudprofle %s", cloudprofile)
	}

	versions := make([]*semver.Version, len(rawVersions))
	for i, rawVersion := range rawVersions {
		v, err := semver.NewVersion(rawVersion)
		if err == nil {
			versions[i] = v
		}
	}
	sort.Sort(semver.Collection(versions))

	return versions[len(versions)-1].String(), nil
}

func getCloudproviderVersions(profile *gardenv1beta1.CloudProfile, cloudprovider string) ([]string, error) {

	switch gardenv1beta1.CloudProvider(cloudprovider) {
	case gardenv1beta1.CloudProviderAWS:
		return profile.Spec.AWS.Constraints.Kubernetes.Versions, nil
	case gardenv1beta1.CloudProviderGCP:
		return profile.Spec.GCP.Constraints.Kubernetes.Versions, nil
	case gardenv1beta1.CloudProviderAzure:
		return profile.Spec.Azure.Constraints.Kubernetes.Versions, nil
	case gardenv1beta1.CloudProviderOpenStack:
		return profile.Spec.OpenStack.Constraints.Kubernetes.Versions, nil
	case gardenv1beta1.CloudProviderAlicloud:
		return profile.Spec.Alicloud.Constraints.Kubernetes.Versions, nil
	default:
		return nil, fmt.Errorf("Unsupported cloudprovider %s", cloudprovider)
	}
}

func addBOMLocationsToTestrun(tr *tmv1beta1.Testrun, componenets []*componentdescriptor.Component) {
	if tr == nil || componenets == nil {
		return
	}
	for _, component := range componenets {
		tr.Spec.TestLocations = append(tr.Spec.TestLocations, tmv1beta1.TestLocation{
			Type:     tmv1beta1.LocationTypeGit,
			Repo:     fmt.Sprintf("https://%s", component.Name),
			Revision: component.Version,
		})
	}
}

func addAnnotationsToTestrun(tr *tmv1beta1.Testrun, annotations map[string]string) {
	if tr == nil {
		return
	}
	tr.Annotations = utils.MergeStringMaps(tr.Annotations, annotations)
}
