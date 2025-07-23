// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
package componentdescriptor

import (
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/compdesc"
	"ocm.software/ocm/api/ocm/extensions/accessmethods/github"
)

// Component describes a component consisting of the git repository url and a version.
type Component struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	SourceRepoURL  string `json:"sourceRepoUrl"`
	SourceRevision string `json:"sourceRevision"`
}

func NewFromVersionedElement(octx ocm.Context, e compdesc.ComponentDescriptor) *Component {
	sources := e.GetSources()
	for _, source := range sources {
		var label compdesc.Label
		exists, _ := source.Labels.GetValue("cloud.gardener/cicd/source", label)
		if exists {
			access := source.GetAccess()
			accessSpec, err := octx.AccessSpecForSpec(access)
			if err == nil {
				ghacces, ok := accessSpec.(*github.AccessSpec)
				if ok {
					return &Component{
						Name:           e.GetName(),
						Version:        e.GetVersion(),
						SourceRepoURL:  ghacces.RepoURL,
						SourceRevision: source.Version,
					}
				}
			}
		}
	}

	return &Component{
		Name:    e.GetName(),
		Version: e.GetVersion(),
	}
}

// ComponentList is a set of multiple components.
type ComponentList []*Component

// ComponentJSON is the object that is written to the elastic search metadata.
type ComponentJSON struct {
	Version string `json:"version"`
}
type components struct {
	components   []*Component
	componentSet map[Component]bool
}
