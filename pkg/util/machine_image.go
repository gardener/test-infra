package util

import (
	"fmt"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	versionutils "github.com/gardener/gardener/pkg/utils/version"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
	"k8s.io/utils/strings/slices"

	"github.com/gardener/test-infra/pkg/common"
)

// GetMachineImageVersion returns a version identified by a pattern or defaults to the version string handed over
func GetMachineImageVersion(cloudprofile gardencorev1beta1.CloudProfile, worker *gardencorev1beta1.Worker) (gardencorev1beta1.MachineImageVersion, error) {
	var (
		machineImageVersion gardencorev1beta1.MachineImageVersion
		version             = *worker.Machine.Image.Version
		arch                = *worker.Machine.Architecture
		imageName           = worker.Machine.Image.Name
		updateStrategy      = ptr.Deref(worker.UpdateStrategy, "")
		inPlace             = updateStrategy == gardencorev1beta1.AutoInPlaceUpdate || updateStrategy == gardencorev1beta1.ManualInPlaceUpdate
		err                 error
	)

	switch version {
	default:
		machineImageVersion = gardencorev1beta1.MachineImageVersion{
			ExpirableVersion: gardencorev1beta1.ExpirableVersion{
				Version: version,
			},
		}
	case common.PatternLatest:
		machineImageVersion, err = GetLatestMachineImageVersion(cloudprofile, imageName, arch, inPlace)
	case common.PatternOneMajorBeforeLatest:
		machineImageVersion, err = GetXMajorsBeforeLatestMachineImageVersion(cloudprofile, imageName, arch, 1, inPlace)
	case common.PatternTwoMajorBeforeLatest:
		machineImageVersion, err = GetXMajorsBeforeLatestMachineImageVersion(cloudprofile, imageName, arch, 2, inPlace)
	case common.PatternThreeMajorBeforeLatest:
		machineImageVersion, err = GetXMajorsBeforeLatestMachineImageVersion(cloudprofile, imageName, arch, 3, inPlace)
	case common.PatternFourMajorBeforeLatest:
		machineImageVersion, err = GetXMajorsBeforeLatestMachineImageVersion(cloudprofile, imageName, arch, 4, inPlace)
	}

	return machineImageVersion, err
}

// GetXMajorsBeforeLatestMachineImageVersion extracts the latest-x major version from a list of relevant versions found in a cloudprofile
func GetXMajorsBeforeLatestMachineImageVersion(cloudprofile gardencorev1beta1.CloudProfile, imageName, arch string, x uint64, inPlace bool) (gardencorev1beta1.MachineImageVersion, error) {
	machineImage, err := GetMachineImage(cloudprofile, imageName)
	if err != nil {
		return gardencorev1beta1.MachineImageVersion{}, err
	}
	machineVersions := machineImage.Versions
	if len(machineVersions) == 0 {
		return gardencorev1beta1.MachineImageVersion{}, fmt.Errorf("no machine image versions found for cloudprofle %s", cloudprofile.GetName())
	}

	machineVersions = FilterExpiredMachineImageVersions(FilterArchSpecificMachineImage(machineVersions, arch))
	if inPlace {
		machineVersions = FilterInPlaceMachineImageVersions(machineVersions)
	}

	return getXMajorsBeforeLatestMachineImageVersion(machineVersions, x)
}

// getXMajorsBeforeLatestMachineImageVersion returns the latest-x version of a list of machine image versions
func getXMajorsBeforeLatestMachineImageVersion(rawVersions []gardencorev1beta1.MachineImageVersion, x uint64) (gardencorev1beta1.MachineImageVersion, error) {
	if len(rawVersions) == 0 {
		return gardencorev1beta1.MachineImageVersion{}, errors.New("no machine image versions found")
	}

	parsedVersions := make([]*semver.Version, 0)
	for _, raw := range rawVersions {
		v, err := semver.NewVersion(raw.Version)
		if err != nil {
			return gardencorev1beta1.MachineImageVersion{}, err
		}
		if v.Metadata() != "" {
			continue
		}
		parsedVersions = append(parsedVersions, v)
	}
	sort.Sort(sort.Reverse(semver.Collection(parsedVersions)))

	xMajorBeforeLatest := x
	cmpVersion := parsedVersions[0]
	for _, version := range parsedVersions {
		if xMajorBeforeLatest == 0 {
			return gardencorev1beta1.MachineImageVersion{
				ExpirableVersion: gardencorev1beta1.ExpirableVersion{
					Version: version.Original(),
				},
			}, nil
		}
		if version.Major() < cmpVersion.Major() {
			xMajorBeforeLatest--
			if xMajorBeforeLatest == 0 {
				return gardencorev1beta1.MachineImageVersion{
					ExpirableVersion: gardencorev1beta1.ExpirableVersion{
						Version: version.Original(),
					},
				}, nil
			}
			cmpVersion = version
		}
	}
	return gardencorev1beta1.MachineImageVersion{}, errors.New(fmt.Sprintf("no machine image version matching the pattern latest-%d found", x))
}

// GetLatestMachineImageVersion returns the latest available machine image version from the cloudprofile
func GetLatestMachineImageVersion(cloudprofile gardencorev1beta1.CloudProfile, imageName, arch string, inPlace bool) (gardencorev1beta1.MachineImageVersion, error) {
	return GetXMajorsBeforeLatestMachineImageVersion(cloudprofile, imageName, arch, 0, inPlace)
}

// FilterArchSpecificMachineImage removes all version which doesn't support given architecture.
func FilterArchSpecificMachineImage(versions []gardencorev1beta1.MachineImageVersion, architecture string) []gardencorev1beta1.MachineImageVersion {
	filtered := make([]gardencorev1beta1.MachineImageVersion, 0)
	for _, v := range versions {
		if slices.Contains(v.Architectures, architecture) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// FilterInPlaceMachineImageVersions filters the machine image versions to only include those that support in-place updates.
func FilterInPlaceMachineImageVersions(versions []gardencorev1beta1.MachineImageVersion) []gardencorev1beta1.MachineImageVersion {
	filtered := make([]gardencorev1beta1.MachineImageVersion, 0)
	for _, v := range versions {
		if v.InPlaceUpdates != nil && v.InPlaceUpdates.Supported {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// FilterExpiredMachineImageVersions removes all expired versions from the list.
func FilterExpiredMachineImageVersions(versions []gardencorev1beta1.MachineImageVersion) []gardencorev1beta1.MachineImageVersion {
	filtered := make([]gardencorev1beta1.MachineImageVersion, 0)
	for _, v := range versions {
		if v.ExpirationDate == nil || v.ExpirationDate.After(time.Now()) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func GetMachineImage(cloudprofile gardencorev1beta1.CloudProfile, imageName string) (gardencorev1beta1.MachineImage, error) {
	for _, image := range cloudprofile.Spec.MachineImages {
		if image.Name == imageName {
			return image, nil
		}
	}
	return gardencorev1beta1.MachineImage{}, fmt.Errorf("no image %s defined in the cloudprofile %s", imageName, cloudprofile.GetName())
}

// GetLatestPreviousVersionForInPlaceUpdate determines the latest previous machine image version
// that supports in-place updates to the current version. It filters the machine image versions
// based on architecture, expiration, and in-place update compatibility, and ensures the version
// satisfies the minimum version requirement for in-place updates.
func GetLatestPreviousVersionForInPlaceUpdate(cloudprofile gardencorev1beta1.CloudProfile, currentMachineImage gardencorev1beta1.ShootMachineImage, arch string) (string, error) {
	machineImage, err := GetMachineImage(cloudprofile, currentMachineImage.Name)
	if err != nil {
		return "", err
	}
	machineVersions := machineImage.Versions
	if len(machineVersions) == 0 {
		return "", fmt.Errorf("no machine image versions found in cloudprofile %s", cloudprofile.GetName())
	}

	machineVersions = FilterExpiredMachineImageVersions(
		FilterInPlaceMachineImageVersions(
			FilterArchSpecificMachineImage(machineVersions, arch),
		),
	)
	if len(machineVersions) == 0 {
		return "", errors.New("no machine image versions found")
	}

	var currentMachineImageVersion *gardencorev1beta1.MachineImageVersion
	for _, version := range machineVersions {
		if validVersion, _ := versionutils.CompareVersions(version.Version, "=", *currentMachineImage.Version); validVersion &&
			version.InPlaceUpdates != nil && version.InPlaceUpdates.Supported {
			currentMachineImageVersion = &version
			break
		}
	}

	if currentMachineImageVersion == nil || currentMachineImageVersion.InPlaceUpdates == nil || !currentMachineImageVersion.InPlaceUpdates.Supported {
		return "", errors.New("specified machine image version is not found in the cloudprofile or does not support in-place updates")
	}
	if currentMachineImageVersion.InPlaceUpdates.MinVersionForUpdate == nil {
		return "", errors.New("current machine image version does not have a minimum version for in-place updates")
	}

	parsedVersions := make([]*semver.Version, 0)
	for _, machineVersion := range machineVersions {
		v, err := semver.NewVersion(machineVersion.Version)
		if err != nil {
			return "", err
		}
		if v.Metadata() != "" {
			continue
		}

		if machineVersion.InPlaceUpdates != nil && machineVersion.InPlaceUpdates.Supported {
			if validVersion, _ := versionutils.CompareVersions(machineVersion.Version, ">=", *currentMachineImage.Version); validVersion {
				continue
			}
			if validVersion, _ := versionutils.CompareVersions(machineVersion.Version, ">=", *currentMachineImageVersion.InPlaceUpdates.MinVersionForUpdate); validVersion {
				parsedVersions = append(parsedVersions, v)
			}
		}
	}
	sort.Sort(sort.Reverse(semver.Collection(parsedVersions)))

	if len(parsedVersions) == 0 {
		return "", errors.New("no machine image versions found that can be in-place updated to the current version")
	}

	return parsedVersions[0].Original(), nil
}
