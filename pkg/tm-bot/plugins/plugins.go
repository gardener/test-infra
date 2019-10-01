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

package plugins

import (
	"fmt"
	"sync"

	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/spf13/pflag"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
)

// Plugin specifies a tm github bot plugin/command that can be triggered by a user
type Plugin interface {
	// Command returns the unique matching command for the plugin
	Command() string

	// Authorization returns the authorization type of the plugin.
	// Defines who is allowed to call the plugin
	Authorization() github.AuthorizationType

	// Description returns a short description of the plugin
	Description() string

	// Example returns an example for the command
	Example() string

	// Flags return command line style flags for the command
	Flags() *pflag.FlagSet

	// Create a deep copy of the plugin
	New(runID string) Plugin

	// Run runs the command with the parsed flags (flag.Parse()) and the event that triggered the command
	Run(fs *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error

	// resume the plugin execution from a persisted state
	ResumeFromState(client github.Client, event *github.GenericRequestEvent, state string) error
}

// Persistence describes the interface for persisting plugin states
type Persistence interface {
	Save(map[string]map[string]*State) error
	Load() (map[string]map[string]*State, error)
}

var Plugins = &plugins{
	registered: make(map[string]Plugin),
	stateMutex: sync.Mutex{},
	states:     make(map[string]map[string]*State),
}

type plugins struct {
	log         logr.Logger
	persistence Persistence
	registered  map[string]Plugin
	stateMutex  sync.Mutex
	states      map[string]map[string]*State
}

// State describes the configuration of a running plugin that is can resume at any time
type State struct {
	Event  *github.GenericRequestEvent
	Custom string
}

// Register registers a plugin with its command to be executed on a event
func Register(plugin Plugin) {
	Plugins.registered[plugin.Command()] = plugin
}

// Setup sets up the plugins with a logger and a persistent storage
func Setup(log logr.Logger, persistence Persistence) {
	Plugins.log = log
	Plugins.persistence = persistence
}

// ResumePlugins resumes all states that can be found in the persistent storage
func ResumePlugins(ghMgr github.Manager) error {
	return Plugins.resumePlugins(ghMgr)
}

func (p *plugins) Get(name string) (string, Plugin, error) {
	plugin, ok := Plugins.registered[name]
	if !ok {
		return "", nil, pluginerr.New(fmt.Sprintf("no plugin found for %s", name), fmt.Sprintf("no plugin found for %s", name))
	}
	runID := util.RandomString(5)
	return runID, plugin.New(runID), nil
}

func (p *plugins) resumePlugins(ghMgr github.Manager) error {
	if p.persistence == nil {
		return nil
	}
	states, err := p.persistence.Load()
	if err != nil {
		return err
	}

	for name, pluginStates := range states {
		for runID, state := range pluginStates {
			go p.resumePlugin(ghMgr, name, runID, state)
		}
	}

	p.states = states
	return nil
}

// initState initializes the default state of a running plugin consisting of the plugins runID and the event
func (p *plugins) initState(pl Plugin, runID string, event *github.GenericRequestEvent) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()

	if p.states == nil {
		p.states = make(map[string]map[string]*State)
	}

	if len(p.states[pl.Command()]) == 0 {
		p.states[pl.Command()] = make(map[string]*State)
	}

	p.states[pl.Command()][runID] = &State{
		Event: event,
	}
}

// UpdateState updates the state of a running plugin and persists the changes
func (p *plugins) UpdateState(pl Plugin, runID string, customState string) error {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()

	state, ok := p.states[pl.Command()][runID]
	if !ok {
		return fmt.Errorf("unknown state for plugin %s with id %s", pl.Command(), runID)
	}
	state.Custom = customState

	if p.persistence != nil {
		if err := p.persistence.Save(p.states); err != nil {
			p.log.Error(err, "unable to persist states")
		}
		p.log.V(3).Info("state persisted")
	}
	return nil
}

// RemoveState removes the state of a running plugin from the persistence
func (p *plugins) RemoveState(pl Plugin, runID string) {
	p.stateMutex.Lock()
	defer p.stateMutex.Unlock()
	if _, ok := p.states[pl.Command()]; !ok {
		return
	}
	delete(p.states[pl.Command()], runID)
	if p.persistence != nil {
		if err := p.persistence.Save(p.states); err != nil {
			p.log.Error(err, "unable to persist states")
		}
		p.log.V(3).Info("state persisted")
	}
}
