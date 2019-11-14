// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gardensetup

import (
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/gardener/test-infra/pkg/common"
	"k8s.io/helm/pkg/strvals"
	"reflect"
	"strings"
)

func ParseFlag(value string) (common.GSExtensions, error) {
	pVals, err := strvals.Parse(value)
	if err != nil {
		return nil, err
	}

	extensions := make(common.GSExtensions, len(pVals))
	for name, val := range pVals {
		var pair []string
		switch v := val.(type) {
		case string:
			pair = strings.Split(v, "::")
		default:
			return nil, fmt.Errorf("unsupported type %s at %s expected string with repo::version pair", reflect.TypeOf(v), name)
		}

		if len(pair) != 2 {
			return nil, fmt.Errorf("value %s of %s has to be of type repo::version", val, name)
		}

		config, err := parseExtensionFromPair(pair)
		if err != nil {
			return nil, err
		}

		extensions[name] = config
	}

	return extensions, nil
}

func parseExtensionFromPair(pair []string) (common.GSExtensionConfig, error) {
	var (
		repository = pair[0]
		revision   = pair[1]
	)
	config := common.GSExtensionConfig{
		Repository: repository,
	}

	// check if revision is a commit sha by checking for length of exactly 40
	if len(revision) == 40 {
		config.Commit = revision
		config.ImageTag = revision
	} else if _, err := semver.NewVersion(revision); err == nil {
		config.Tag = revision
	} else {
		config.Branch = revision
	}

	return config, nil
}

// MergeExtensions merges gardener extensions whereas new will overwrite all keys that are defined by base
func MergeExtensions(base, newVal common.GSExtensions) common.GSExtensions {
	for key, val := range newVal {
		base[key] = val
	}
	return base
}

// ConvertRawDependenciesToInternalExtensionConfig converts gardener dependencies to gardensetup extension configuration that can be used in the acre.yaml
func ConvertRawDependenciesToInternalExtensionConfig(deps map[string]common.GSVersion) common.GSExtensions {
	extensions := make(common.GSExtensions, len(deps))
	for name, cfg := range deps {
		extensions[name] = common.GSExtensionConfig{
			Tag:        cfg.Version,
			Repository: cfg.Repository,
		}
	}
	return extensions
}
