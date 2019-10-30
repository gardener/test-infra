package util

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver"
	gardenv1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
)

// GetK8sVersions returns all K8s version that should be rendered by the chart
func GetK8sVersions(k8sClient kubernetes.Interface, config common.ShootKubernetesVersionFlavor, cloudprofile string) ([]gardenv1alpha1.ExpirableVersion, error) {
	if config.Versions != nil && len(*config.Versions) != 0 {
		return *config.Versions, nil
	}
	if config.Pattern == nil {
		return nil, errors.New("no kubernetes versions or patterns are defined")
	}
	pattern := *config.Pattern

	// if the pattern is "latest" get the latest k8s version
	if pattern == common.PatternLatest {
		version, err := GetLatestK8sVersion(k8sClient, cloudprofile)
		if err != nil {
			return nil, err
		}
		return []gardenv1alpha1.ExpirableVersion{version}, nil
	}

	versions, err := GetK8sVersionsFromCloudprofile(k8sClient, cloudprofile)
	if err != nil {
		return nil, err
	}

	constraint, err := semver.NewConstraint(pattern)
	if err != nil {
		return nil, err
	}

	filtered := make([]gardenv1alpha1.ExpirableVersion, 0)
	for _, expirableVersion := range versions {
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

	return filtered, nil
}

func GetK8sVersionsFromCloudprofile(k8sClient kubernetes.Interface, cloudprofile string) ([]gardenv1alpha1.ExpirableVersion, error) {
	ctx := context.Background()
	defer ctx.Done()

	profile := &gardenv1alpha1.CloudProfile{}
	err := k8sClient.Client().Get(ctx, types.NamespacedName{Name: cloudprofile}, profile)
	if err != nil {
		return nil, err
	}
	if len(profile.Spec.Kubernetes.Versions) == 0 {
		return nil, fmt.Errorf("no kubernetes versions found for cloudprofile %s", cloudprofile)
	}
	return profile.Spec.Kubernetes.Versions, nil
}

// GetLatestK8sVersion returns the lates avilable kubernetes version from the cloudprofile
func GetLatestK8sVersion(k8sClient kubernetes.Interface, cloudprofile string) (gardenv1alpha1.ExpirableVersion, error) {
	rawVersions, err := GetK8sVersionsFromCloudprofile(k8sClient, cloudprofile)
	if err != nil {
		return gardenv1alpha1.ExpirableVersion{}, err
	}

	if len(rawVersions) == 0 {
		return gardenv1alpha1.ExpirableVersion{}, fmt.Errorf("no kubernetes versions found for cloudprofle %s", cloudprofile)
	}

	return GetLatestVersion(rawVersions)
}
