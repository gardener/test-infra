// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package componentdescriptor

import (
	"github.com/open-component-model/ocm/pkg/common"
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
