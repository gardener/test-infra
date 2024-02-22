// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"context"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
)

// HandleRequest parses a github event and executes the found Plugins
func HandleRequest(client github.Client, event *github.GenericRequestEvent) error {
	return plugins.HandleRequest(client, event)
}

// HandleRequest parses a github event and executes the found Plugins
func (p *Plugins) HandleRequest(client github.Client, event *github.GenericRequestEvent) error {
	commands, err := ParseCommands(event.GetMessage())
	if err != nil {
		return pluginerr.Wrap(err, "Internal parse error")
	}

	for _, args := range commands {
		p.runPlugin(client, event, args)
	}

	return nil
}

// runPlugin runs a plugin with a event and its arguments
func (p *Plugins) runPlugin(client github.Client, event *github.GenericRequestEvent, args []string) {
	runID, plugin, err := p.Get(args[0])
	if err != nil {
		p.log.Error(err, "unable to get plugin for command", "command", args[0])
		return
	}

	if !client.IsAuthorized(plugin.Authorization(), event) {
		p.log.V(3).Info("user not authorized", "user", event.GetAuthorName(), "plugin", plugin.Command())
		_, _ = client.Comment(context.TODO(), event, FormatUnauthorizedResponse(event.GetAuthorName(), args[0]))
		return
	}

	p.initState(plugin, runID, event)

	fs := plugin.Flags()
	if err := fs.Parse(args[1:]); err != nil {
		p.RemoveState(plugin, runID)
		_ = p.Error(client, event, plugin, pluginerr.New(err.Error(), "unable to parse flags"))
		return
	}
	if err := plugin.Run(fs, client, event); err != nil {
		if !pluginerr.IsRecoverable(err) {
			plugins.RemoveState(plugin, runID)
			_ = p.Error(client, event, plugin, err)
		}
		return
	}

	plugins.RemoveState(plugin, runID)
}

// resumePlugin resumes a plugin from its previously written state
func (p *Plugins) resumePlugin(ghMgr github.Manager, name, runID string, state *State) {
	ghClient, err := ghMgr.GetClient(state.Event)
	if err != nil {
		p.log.Error(err, "unable to get github client for state", "plugin", name)
		return
	}

	_, plugin, err := p.Get(name)
	if err != nil {
		p.log.Error(err, "unable to get plugin for state", "plugin", name)
		return
	}
	plugin = plugin.New(runID)

	if err := plugin.ResumeFromState(ghClient, state.Event, state.Custom); err != nil {
		if !pluginerr.IsRecoverable(err) {
			p.RemoveState(plugin, runID)
			_ = p.Error(ghClient, state.Event, plugin, err)
		}
		return
	}
	p.RemoveState(plugin, runID)
}

// Error responds to the client if an error occurs
func (p *Plugins) Error(client github.Client, event *github.GenericRequestEvent, plugin Plugin, err error) error {
	p.log.Error(err, err.Error())
	pluginErr, ok := err.(*pluginerr.PluginError)
	if !ok {
		return nil
	}

	detailedMsg := FormatUsageError(plugin.Command(), plugin.Description(), plugin.Example(), plugin.Flags().FlagUsages())
	if !pluginerr.OmitLongMessage(err) {
		detailedMsg = pluginErr.LongMsg
	}

	if plugin != nil {
		_, err := client.Comment(context.TODO(), event, FormatErrorResponse(event.GetAuthorName(), pluginerr.ShortForError(err), detailedMsg))
		return err
	}
	_, err = client.Comment(context.TODO(), event, FormatSimpleErrorResponse(event.GetAuthorName(), pluginerr.ShortForError(err)))
	return err
}
