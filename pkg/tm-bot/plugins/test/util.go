// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package test

import (
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	ghutil "github.com/gardener/test-infra/pkg/tm-bot/github"
)

// InjectRepositoryLocation adds the current repository and its branch to the default locationset
func InjectRepositoryLocation(event *ghutil.GenericRequestEvent, tr *tmv1beta1.Testrun) error {
	location := tmv1beta1.TestLocation{
		Type:     tmv1beta1.LocationTypeGit,
		Repo:     event.Repository.GetCloneURL(),
		Revision: event.Head,
	}

	if len(tr.Spec.LocationSets) == 0 {
		tr.Spec.LocationSets = []tmv1beta1.LocationSet{
			{
				Name:      "default",
				Default:   true,
				Locations: []tmv1beta1.TestLocation{location},
			},
		}
		return nil
	}

	// find default location
	for i, ls := range tr.Spec.LocationSets {
		if ls.Default {
			tr.Spec.LocationSets[i].Locations = append(ls.Locations, location)
			return nil
		}
	}

	// no default location could be found add as new
	tr.Spec.LocationSets = append(tr.Spec.LocationSets, tmv1beta1.LocationSet{
		Name:      "default",
		Default:   true,
		Locations: []tmv1beta1.TestLocation{location},
	})

	return nil
}
