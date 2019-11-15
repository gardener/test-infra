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

package ghval

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
)

// Type represents the stored type of StringOrGitHubValue.
type Type int

const (
	String  Type = iota // The IntOrString holds an string.
	GHValue             // The IntOrString holds a GitHubValue.
)

type StringOrGitHubValue struct {
	Type     Type
	StrValue string
	GHValue  GitHubValue
}

// GitHubValue describes a value that will be determined during runtime by
// - reading a specific file from a path
// - get the current PR's commit hash
type GitHubValue struct {
	// raw value
	Value *string `json:"value"`

	// Path will read the value from the default branch
	Path *string `json:"path"`

	// StructuredJSONPath reads the specified path from the parsed path file.
	// Path has to be defined and has to be in yaml format in order to get the path.
	StructuredJSONPath *string `json:"structuredJSONPath"`

	// Use the commit of the current Pull Request
	PRHead *bool `json:"prHead"`
}

func (v *StringOrGitHubValue) UnmarshalJSON(value []byte) (err error) {
	if value[0] == '"' {
		v.Type = String
		return json.Unmarshal(value, &v.StrValue)
	}
	if value[0] == '{' {
		v.Type = GHValue
		return json.Unmarshal(value, &v.GHValue)
	}
	return errors.New("unknown type")
}

// MarshalJSON implements the json.Marshaller interface.
func (v *StringOrGitHubValue) MarshalJSON() ([]byte, error) {
	switch v.Type {
	case String:
		return json.Marshal(v.StrValue)
	case GHValue:
		return json.Marshal(v.GHValue)
	default:
		return []byte{}, fmt.Errorf("impossible StringOrGitHubValue.Type")
	}
}

func (v *StringOrGitHubValue) Value() *GitHubValue {
	if v.Type == String {
		return &GitHubValue{
			Value: &v.StrValue,
		}
	}
	return &v.GHValue
}
