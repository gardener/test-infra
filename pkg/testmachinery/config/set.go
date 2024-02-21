// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"

func NewSet(elements ...*Element) Set {
	s := make(Set)
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
	newSet := make(Set)
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
