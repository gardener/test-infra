// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0
package componentdescriptor

import (
	common "ocm.software/ocm/api/utils/misc"
)

// Component describes a component consisting of the git repository url and a version.
type Component struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewFromVersionedElement(e common.VersionedElement) *Component {
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
