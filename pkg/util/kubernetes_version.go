package util

import (
	"fmt"
	"github.com/Masterminds/semver"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/pkg/errors"
)

// GetK8sVersions returns all K8s version that should be rendered by the chart
func GetK8sVersions(cloudprofile gardencorev1beta1.CloudProfile, config common.ShootKubernetesVersionFlavor, filterPatchVersions bool) ([]gardencorev1beta1.ExpirableVersion, error) {
	if config.Versions != nil && len(*config.Versions) != 0 {
		return *config.Versions, nil
	}
	if config.Pattern == nil {
		return nil, errors.New("no kubernetes versions or patterns are defined")
	}
	pattern := *config.Pattern

	// if the pattern is "latest" get the latest k8s version
	if pattern == common.PatternLatest {
		version, err := GetLatestK8sVersion(cloudprofile)
		if err != nil {
			return nil, err
		}
		return []gardencorev1beta1.ExpirableVersion{version}, nil
	}

	constraint, err := semver.NewConstraint(pattern)
	if err != nil {
		return nil, err
	}

	filtered := make([]gardencorev1beta1.ExpirableVersion, 0)
	for _, expirableVersion := range FilterExpiredVersions(cloudprofile.Spec.Kubernetes.Versions) {
		version, err := semver.NewVersion(expirableVersion.Version)
		if err != nil {
			return nil, err
		}
		if constraint.Check(version) {
			filtered = append(filtered, expirableVersion)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no K8s version can be specified")
	}

	if (config.FilterPatchVersions != nil && *config.FilterPatchVersions) || (config.FilterPatchVersions == nil && filterPatchVersions) {
		return FilterPatchVersions(filtered)
	}

	return filtered, nil
}

// GetPreviousKubernetesVersions returns the 2 latest previous minor patch versions.
func GetPreviousKubernetesVersions(cloudprofile gardencorev1beta1.CloudProfile, currentVersion gardencorev1beta1.ExpirableVersion) (gardencorev1beta1.ExpirableVersion, gardencorev1beta1.ExpirableVersion, error) {
	type versionWrapper struct {
		expirableVersion gardencorev1beta1.ExpirableVersion
		semverVersion    *semver.Version
	}
	currentSemver, err := semver.NewVersion(currentVersion.Version)
	if err != nil {
		return currentVersion, currentVersion, err
	}
	prevBaseVersion, err := DecMinorVersion(currentSemver)
	if err != nil {
		return currentVersion, currentVersion, err
	}
	prevMinorConstraint, err := semver.NewConstraint(fmt.Sprintf("~%s", prevBaseVersion.String()))
	if err != nil {
		return currentVersion, currentVersion, err
	}

	var (
		prevPatch    *versionWrapper
		prevPrePatch *versionWrapper
	)

	for _, expirableVersion := range FilterExpiredVersions(cloudprofile.Spec.Kubernetes.Versions) {
		version, err := semver.NewVersion(expirableVersion.Version)
		if err != nil {
			return currentVersion, currentVersion, err
		}
		if !prevMinorConstraint.Check(version) {
			continue
		}
		if prevPatch == nil || version.GreaterThan(prevPatch.semverVersion) {
			prevPrePatch = prevPatch
			prevPatch = &versionWrapper{
				expirableVersion: expirableVersion,
				semverVersion:    version,
			}
			continue
		}
		if prevPrePatch == nil || version.GreaterThan(prevPrePatch.semverVersion) {
			prevPrePatch = &versionWrapper{
				expirableVersion: expirableVersion,
				semverVersion:    version,
			}
		}
	}

	if prevPatch == nil {
		prevPatch = &versionWrapper{
			expirableVersion: currentVersion,
			semverVersion:    currentSemver,
		}
	}
	if prevPrePatch == nil {
		prevPrePatch = prevPatch
	}

	return prevPrePatch.expirableVersion, prevPatch.expirableVersion, nil
}

// GetLatestK8sVersion returns the lates avilable kubernetes version from the cloudprofile
func GetLatestK8sVersion(cloudprofile gardencorev1beta1.CloudProfile) (gardencorev1beta1.ExpirableVersion, error) {
	if len(cloudprofile.Spec.Kubernetes.Versions) == 0 {
		return gardencorev1beta1.ExpirableVersion{}, fmt.Errorf("no kubernetes versions found for cloudprofle %s", cloudprofile.Name)
	}

	return GetLatestVersion(FilterExpiredVersions(cloudprofile.Spec.Kubernetes.Versions))
}
