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
	"bufio"
	"github.com/gardener/test-infra/pkg/tm-bot/github"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"io"
	"strings"
)

// HandleRequest parses a github event and executes the found plugins
func HandleRequest(client github.Client, event *github.GenericRequestEvent) error {
	commands, err := ParseCommands(event.GetMessage())
	if err != nil {
		return pluginerr.Wrap(err, "Internal parse error")
	}

	for _, args := range commands {
		go Plugins.runPlugin(client, event, args)
	}

	return nil
}

// runPlugin runs a plugin with a event and its arguments
func (p *plugins) runPlugin(client github.Client, event *github.GenericRequestEvent, args []string) {
	runID, plugin, err := p.Get(args[0])
	if err != nil {
		_ = p.Error(client, event, nil, err)
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
			Plugins.RemoveState(plugin, runID)
			_ = p.Error(client, event, plugin, err)
		}
		return
	}

	Plugins.RemoveState(plugin, runID)
}

// resumePlugin resumes a plugin from its previously written state
func (p *plugins) resumePlugin(ghMgr github.Manager, name, runID string, state *State) {
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
	Plugins.RemoveState(plugin, runID)
}

// Error responds to the client if an error occurs
func (p *plugins) Error(client github.Client, event *github.GenericRequestEvent, plugin Plugin, err error) error {
	p.log.Error(err, err.Error())

	if plugin != nil {
		_, err := client.Comment(event, FormatErrorResponse(event.GetAuthorName(), pluginerr.ShortForError(err), FormatUsageError(plugin.Command(), plugin.Description(), plugin.Example(), plugin.Flags().FlagUsages())))
		return err
	}
	_, err = client.Comment(event, FormatSimpleErrorResponse(event.GetAuthorName(), pluginerr.ShortForError(err)))
	return err
}

// ParseCommands parses a message and returns a string of commands and arguments
func ParseCommands(message string) ([][]string, error) {
	r := bufio.NewReader(strings.NewReader(message))
	var (
		commands = make([][]string, 0)
		line     string
		err      error
	)
	for {
		line, err = r.ReadString('\n')

		trimmedLine := strings.Trim(line, " \n\t")
		if strings.HasPrefix(trimmedLine, "/") {
			args := strings.Split(trimmedLine, " ")
			args[0] = strings.TrimPrefix(args[0], "/")
			commands = append(commands, args)
		}

		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
	}

	return commands, nil
}
