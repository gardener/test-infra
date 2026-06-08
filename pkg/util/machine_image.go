package util

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	versionutils "github.com/gardener/gardener/pkg/utils/version"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"

	"github.com/gardener/test-infra/pkg/common"
)

// providerConfigMachineImages models the subset of
// cloudprofile.Spec.ProviderConfig that carries machine-image architecture
// info. The capabilities format is the target schema; the region and flat
// formats are legacy and can be dropped once all providers have migrated.
type providerConfigMachineImages struct {
	MachineImages []providerConfigMachineImage `json:"machineImages"`
}

type providerConfigMachineImage struct {
	Name     string                              `json:"name"`
	Versions []providerConfigMachineImageVersion `json:"versions"`
}

type providerConfigMachineImageVersion struct {
	Version string `json:"version"`

	// Capabilities format (target schema):
	//   versions[].capabilityFlavors[].capabilities.architecture
	CapabilityFlavors []providerConfigCapabilityFlavor `json:"capabilityFlavors,omitempty"`

	// --- Legacy formats: remove once migration to capability flavors is complete. ---
	// Flat format:   versions[].architecture (singular; version may repeat per arch)
	Architecture string `json:"architecture,omitempty"`
	// Region format: versions[].regions[].architecture
	Regions []providerConfigRegion `json:"regions,omitempty"`
	// --- End legacy formats. ---
}

type providerConfigRegion struct {
	Architecture string `json:"architecture"`
}

type providerConfigCapabilityFlavor struct {
	Capabilities providerConfigCapabilities `json:"capabilities"`
}

type providerConfigCapabilities struct {
	Architecture []string `json:"architecture"`
}

// FilterMachineImageVersionsByArch returns versions of imageName that support
// the given architecture, based on cloudprofile.Spec.ProviderConfig. If
// cloudprofile.Spec.MachineCapabilities defines "architecture" with exactly
// one value, every version is considered to support that single arch and the
// providerConfig is not consulted.
func FilterMachineImageVersionsByArch(cloudprofile gardencorev1beta1.CloudProfile, imageName string, versions []gardencorev1beta1.MachineImageVersion, architecture string) []gardencorev1beta1.MachineImageVersion {
	if implicitArch := singleValuedArchitectureCapability(cloudprofile); implicitArch != "" {
		if implicitArch != architecture {
			return []gardencorev1beta1.MachineImageVersion{}
		}
		filtered := make([]gardencorev1beta1.MachineImageVersion, len(versions))
		copy(filtered, versions)
		return filtered
	}

	supportedVersions := supportedArchsByVersion(cloudprofile, imageName)
	filtered := make([]gardencorev1beta1.MachineImageVersion, 0, len(versions))
	for _, v := range versions {
		if archs, ok := supportedVersions[v.Version]; ok && archs.Has(architecture) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// supportedArchsByVersion returns version -> supported archs for imageName,
// decoded from cloudprofile.Spec.ProviderConfig.
func supportedArchsByVersion(cloudprofile gardencorev1beta1.CloudProfile, imageName string) map[string]sets.Set[string] {
	lookup := make(map[string]sets.Set[string])

	if cloudprofile.Spec.ProviderConfig == nil || len(cloudprofile.Spec.ProviderConfig.Raw) == 0 {
		return lookup
	}

	var decoded providerConfigMachineImages
	if err := json.Unmarshal(cloudprofile.Spec.ProviderConfig.Raw, &decoded); err != nil {
		return lookup
	}

	for _, image := range decoded.MachineImages {
		if image.Name != imageName {
			continue
		}
		for _, v := range image.Versions {
			archs, ok := lookup[v.Version]
			if !ok {
				archs = sets.New[string]()
			}
			archs.Insert(archsFromCapabilityFlavors(v)...)
			archs.Insert(archsFromLegacyFormats(v)...)
			if archs.Len() > 0 {
				lookup[v.Version] = archs
			}
		}
	}
	return lookup
}

// archsFromCapabilityFlavors extracts architectures from the capability-flavor
// (target) format on a single providerConfig version entry.
func archsFromCapabilityFlavors(v providerConfigMachineImageVersion) []string {
	archs := make([]string, 0)
	for _, flavor := range v.CapabilityFlavors {
		archs = append(archs, flavor.Capabilities.Architecture...)
	}
	return archs
}

// archsFromLegacyFormats extracts architectures from the legacy region and
// flat formats on a single providerConfig version entry. Remove this function
// (and its call site) once all providers have migrated to capability flavors.
func archsFromLegacyFormats(v providerConfigMachineImageVersion) []string {
	archs := make([]string, 0)
	for _, region := range v.Regions {
		archs = append(archs, region.Architecture)
	}
	if v.Architecture != "" {
		archs = append(archs, v.Architecture)
	}
	return archs
}

// singleValuedArchitectureCapability returns the sole value of the
// "architecture" entry in cloudprofile.Spec.MachineCapabilities, or "" if it
// is missing or has zero/multiple values.
func singleValuedArchitectureCapability(cloudprofile gardencorev1beta1.CloudProfile) string {
	for _, def := range cloudprofile.Spec.MachineCapabilities {
		if def.Name != "architecture" {
			continue
		}
		if len(def.Values) == 1 {
			return def.Values[0]
		}
		return ""
	}
	return ""
}

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

	machineVersions = FilterExpiredMachineImageVersions(FilterMachineImageVersionsByArch(cloudprofile, imageName, machineVersions, arch))
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
			FilterMachineImageVersionsByArch(cloudprofile, currentMachineImage.Name, machineVersions, arch),
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
