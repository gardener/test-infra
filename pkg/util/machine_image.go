package util

import (
	"context"
	"fmt"
	gardenv1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"k8s.io/apimachinery/pkg/types"
)

// GetLatestK8sVersion returns the latest avilable kubernetes version from the cloudprofile
func GetLatestMachineImageVersion(k8sClient kubernetes.Interface, cloudprofile, imageName string) (gardenv1alpha1.ExpirableVersion, error) {
	rawVersions, err := GetMachineImageVersionsFromCloudprofile(k8sClient, cloudprofile, imageName)
	if err != nil {
		return gardenv1alpha1.ExpirableVersion{}, err
	}

	if len(rawVersions) == 0 {
		return gardenv1alpha1.ExpirableVersion{}, fmt.Errorf("no kubernetes versions found for cloudprofle %s", cloudprofile)
	}

	return GetLatestVersion(rawVersions)
}

func GetMachineImageVersionsFromCloudprofile(k8sClient kubernetes.Interface, cloudprofile, imageName string) ([]gardenv1alpha1.ExpirableVersion, error) {
	ctx := context.Background()
	defer ctx.Done()

	profile := &gardenv1alpha1.CloudProfile{}
	err := k8sClient.Client().Get(ctx, types.NamespacedName{Name: cloudprofile}, profile)
	if err != nil {
		return nil, err
	}
	if len(profile.Spec.MachineImages) == 0 {
		return nil, fmt.Errorf("no machine images found for cloudprofile %s", cloudprofile)
	}
	for _, image := range profile.Spec.MachineImages {
		if image.Name == imageName {
			return image.Versions, nil
		}
	}

	return nil, fmt.Errorf("no image versions found for %s", imageName)
}
