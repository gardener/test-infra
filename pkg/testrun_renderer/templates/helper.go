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

package templates

import (
	"fmt"
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
)

func GetLocationsFromExtensions(extensions common.GSExtensions) []v1beta1.TestLocation {
	extSet := make(map[string]interface{}, 0)
	ext := make([]v1beta1.TestLocation, 0)
	for _, e := range extensions {
		var revision string
		if e.Tag != "" {
			revision = e.Tag
		} else if e.Commit != "" {
			revision = e.Commit
		} else if e.Branch != "" {
			revision = e.Branch
		}
		name := fmt.Sprintf("%s/%s", e.Repository, revision)
		if _, ok := extSet[name]; ok {
			continue
		}

		extSet[name] = new(interface{})
		ext = append(ext, v1beta1.TestLocation{
			Type:     v1beta1.LocationTypeGit,
			Repo:     e.Repository,
			Revision: revision,
		})
	}
	return ext
}
