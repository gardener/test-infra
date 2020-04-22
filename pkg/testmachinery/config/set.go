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

package config

import "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"

func NewSet(elements ...*Element) Set {
	s := make(Set, 0)
	for _, e := range elements {
		s[e.Info.Name] = e
	}
	return s
}

func (s Set) List() []*Element {
	list := make([]*Element, len(s))
	i := 0
	for _, e := range s {
		list[i] = e
		i++
	}
	return list
}

func (s Set) RawList() []*v1beta1.ConfigElement {
	list := make([]*v1beta1.ConfigElement, len(s))
	i := 0
	for _, e := range s {
		list[i] = e.Info
		i++
	}
	return list
}

func (s Set) Copy() Set {
	newSet := make(Set, 0)
	for key, value := range s {
		newSet[key] = value
	}
	return newSet
}

// Set adds an element to the set.
// If the element already exist this element will be overwritten
func (s Set) Set(e *Element) {
	s[e.Info.Name] = e
}

// Add adds elements to the set if the element does not exist
// or the element level is higher than the existing one
func (s Set) Add(elements ...*Element) {
	for _, e := range elements {
		if !s.Has(e) {
			s.Set(e)
			continue
		}
		if s[e.Info.Name].Level < e.Level {
			s.Set(e)
		}
	}
}

// Has checks if the elemnt is already in the set
func (s Set) Has(e *Element) bool {
	_, ok := s[e.Info.Name]
	return ok
}
