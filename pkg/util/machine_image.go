package util

import (
	"fmt"
	"time"

	"github.com/Masterminds/semver"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/pkg/errors"
)

// GetLatestK8sVersion returns the latest avilable kubernetes version from the cloudprofile
func GetLatestMachineImageVersion(cloudprofile gardencorev1beta1.CloudProfile, imageName string) (gardencorev1beta1.MachineImageVersion, error) {
	machineImage, err := GetMachineImage(cloudprofile, imageName)
	if err != nil {
		return gardencorev1beta1.MachineImageVersion{}, err
	}
	machineVersions := machineImage.Versions
	if len(machineVersions) == 0 {
		return gardencorev1beta1.MachineImageVersion{}, fmt.Errorf("no machine image versions found for cloudprofle %s", cloudprofile.GetName())
	}

	return getLatestMachineImageVersion(FilterExpiredMachineImageVersions(machineVersions))
}

// FilterExpiredMachineImageVersions removes all expired versions from the list.
func FilterExpiredMachineImageVersions(versions []gardencorev1beta1.MachineImageVersion) []gardencorev1beta1.MachineImageVersion {
	filtered := make([]gardencorev1beta1.MachineImageVersion, 0)
	for _, v := range versions {
		if v.ExpirationDate == nil || v.ExpirationDate.Time.After(time.Now()) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// getLatestMachineImageVersion returns the latest image from a list of expirable versions
func getLatestMachineImageVersion(rawVersions []gardencorev1beta1.MachineImageVersion) (gardencorev1beta1.MachineImageVersion, error) {
	if len(rawVersions) == 0 {
		return gardencorev1beta1.MachineImageVersion{}, errors.New("no kubernetes versions found")
	}

	var (
		latestExpVersion gardencorev1beta1.MachineImageVersion
		latestVersion    *semver.Version
	)

	for _, rawVersion := range rawVersions {
		v, err := semver.NewVersion(rawVersion.Version)
		if err != nil {
			return gardencorev1beta1.MachineImageVersion{}, err
		}
		if latestVersion == nil || v.GreaterThan(latestVersion) {
			latestVersion = v
			latestExpVersion = rawVersion
		}
	}
	return latestExpVersion, nil
}

func GetMachineImage(cloudprofile gardencorev1beta1.CloudProfile, imageName string) (gardencorev1beta1.MachineImage, error) {
	for _, image := range cloudprofile.Spec.MachineImages {
		if image.Name == imageName {
			return image, nil
		}
	}
	return gardencorev1beta1.MachineImage{}, fmt.Errorf("no image %s defined in the cloudprofile %s", imageName, cloudprofile.GetName())
}
