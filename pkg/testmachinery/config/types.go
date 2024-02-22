// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

type Level int

const (
	LevelTestDefinition Level = 0
	LevelGlobal         Level = 5
	LevelShared         Level = 10
	LevelStep           Level = 15
)

// Element represents a configuration parameter for tests.
type Element struct {
	Info *tmv1beta1.ConfigElement
	Level
	name string
}

// Set is a config element set
type Set map[string]*Element
