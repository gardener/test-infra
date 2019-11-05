package util

import (
	"fmt"
	gardenv1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
)

// GetLatestK8sVersion returns the latest avilable kubernetes version from the cloudprofile
func GetLatestMachineImageVersion(cloudprofile gardenv1alpha1.CloudProfile, imageName string) (gardenv1alpha1.ExpirableVersion, error) {
	machineImage, err := GetMachineImage(cloudprofile, imageName)
	if err != nil {
		return gardenv1alpha1.ExpirableVersion{}, err
	}
	machineVersions := machineImage.Versions
	if len(machineVersions) == 0 {
		return gardenv1alpha1.ExpirableVersion{}, fmt.Errorf("no kubernetes versions found for cloudprofle %s", cloudprofile.GetName())
	}

	return GetLatestVersion(machineVersions)
}

func GetMachineImage(cloudprofile gardenv1alpha1.CloudProfile, imageName string) (gardenv1alpha1.MachineImage, error) {
	for _, image := range cloudprofile.Spec.MachineImages {
		if image.Name == imageName {
			return image, nil
		}
	}
	return gardenv1alpha1.MachineImage{}, fmt.Errorf("no image %s defined in the cloudprofile %s", imageName, cloudprofile.GetName())
}
