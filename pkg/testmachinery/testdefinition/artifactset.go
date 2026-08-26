// SPDX-FileCopyrightText: Copyright Contributors to the Gardener project
//
// SPDX-License-Identifier: Apache-2.0

package testdefinition

type ArtifactSet map[string]empty

type empty struct{}

func (s ArtifactSet) Add(key string) {
	s[key] = empty{}
}

func (s ArtifactSet) Has(key string) bool {
	_, ok := s[key]
	return ok
}

func (s ArtifactSet) Copy() ArtifactSet {
	newSet := make(ArtifactSet, len(s))
	for key, value := range s {
		newSet[key] = value
	}
	return newSet
}
