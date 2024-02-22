// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/gardener/test-infra/pkg/util"
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

	// Config returns description for the default config that can be used in the repository
	Config() string

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

var plugins = &Plugins{
	registered: make(map[string]Plugin),
	stateMutex: sync.Mutex{},
	states:     make(map[string]map[string]*State),
}

type Plugins struct {
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

func New(log logr.Logger, persistence Persistence) *Plugins {
	return &Plugins{
		log:         log,
		persistence: persistence,
		registered:  make(map[string]Plugin),
		stateMutex:  sync.Mutex{},
		states:      make(map[string]map[string]*State),
	}
}

// Setup sets up the Plugins with a logger and a persistent storage
func Setup(log logr.Logger, persistence Persistence) {
	plugins.log = log
	plugins.persistence = persistence
}

// Register registers a plugin with its command to be executed on a event
func Register(plugin Plugin) {
	plugins.Register(plugin)
}

// Register registers a plugin with its command to be executed on a event
func (p *Plugins) Register(plugin Plugin) {
	p.registered[plugin.Command()] = plugin
	p.log.Info("registered plugin", "name", plugin.Command())
}

// Get returns a specific plugin
func Get(name string) (string, Plugin, error) {
	return plugins.Get(name)
}

func (p *Plugins) Get(name string) (string, Plugin, error) {
	plugin, ok := p.registered[name]
	if !ok {
		return "", nil, pluginerr.New(fmt.Sprintf("no plugin found for %s", name), fmt.Sprintf("no plugin found for %s", name))
	}
	runID := util.RandomString(5)
	return runID, plugin.New(runID), nil
}

// GetAll returns all registered plugins
func GetAll() []Plugin {
	pluginSet := sets.New[string]()
	for _, plugin := range plugins.registered {
		pluginSet.Insert(plugin.Command())
	}

	list := make([]Plugin, pluginSet.Len())
	for i, name := range pluginSet.UnsortedList() {
		_, plugin, _ := plugins.Get(name)
		list[i] = plugin
	}

	return list
}

// ResumePlugins resumes all states that can be found in the persistent storage
func ResumePlugins(ghMgr github.Manager) error {
	return plugins.ResumePlugins(ghMgr)
}

// ResumePlugins resumes all states that can be found in the persistent storage
func (p *Plugins) ResumePlugins(ghMgr github.Manager) error {
	var err error
	if p.persistence == nil {
		return nil
	}
	p.states, err = p.persistence.Load()
	if err != nil {
		return err
	}

	for name, pluginStates := range p.states {
		for runID, state := range pluginStates {
			go p.resumePlugin(ghMgr, name, runID, state)
		}
	}
	return nil
}

// initState initializes the default state of a running plugin consisting of the Plugins runID and the event
func (p *Plugins) initState(pl Plugin, runID string, event *github.GenericRequestEvent) {
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

func UpdateState(pl Plugin, runID string, customState string) error {
	return plugins.UpdateState(pl, runID, customState)
}

// UpdateState updates the state of a running plugin and persists the changes
func (p *Plugins) UpdateState(pl Plugin, runID string, customState string) error {
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
func (p *Plugins) RemoveState(pl Plugin, runID string) {
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
