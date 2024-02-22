// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
