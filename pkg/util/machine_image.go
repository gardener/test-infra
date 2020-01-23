package util

import (
	"fmt"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
)

// GetLatestK8sVersion returns the latest avilable kubernetes version from the cloudprofile
func GetLatestMachineImageVersion(cloudprofile gardencorev1beta1.CloudProfile, imageName string) (gardencorev1beta1.ExpirableVersion, error) {
	machineImage, err := GetMachineImage(cloudprofile, imageName)
	if err != nil {
		return gardencorev1beta1.ExpirableVersion{}, err
	}
	machineVersions := machineImage.Versions
	if len(machineVersions) == 0 {
		return gardencorev1beta1.ExpirableVersion{}, fmt.Errorf("no kubernetes versions found for cloudprofle %s", cloudprofile.GetName())
	}

	return GetLatestVersion(FilterExpiredVersions(machineVersions))
}

func GetMachineImage(cloudprofile gardencorev1beta1.CloudProfile, imageName string) (gardencorev1beta1.MachineImage, error) {
	for _, image := range cloudprofile.Spec.MachineImages {
		if image.Name == imageName {
			return image, nil
		}
	}
	return gardencorev1beta1.MachineImage{}, fmt.Errorf("no image %s defined in the cloudprofile %s", imageName, cloudprofile.GetName())
}
