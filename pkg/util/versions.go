package util

import (
	"github.com/Masterminds/semver/v3"
	"github.com/pkg/errors"

	"github.com/gardener/test-infra/pkg/common"
)

// GetLatestVersionFromConstraint returns the latest version that matches a constraint
func GetLatestVersionFromConstraint(versions []*semver.Version, constraintString string) (*semver.Version, error) {
	if len(versions) == 0 {
		return nil, errors.New("no versions are defined")
	}

	if constraintString == common.PatternLatest {
		constraintString = "*"
	}

	constraint, err := semver.NewConstraint(constraintString)
	if err != nil {
		return nil, err
	}

	var matched *semver.Version
	for _, version := range versions {
		if constraint.Check(version) {
			if matched == nil || version.GreaterThan(matched) {
				matched = version
			}
		}
	}

	return matched, nil
}
