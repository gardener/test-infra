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
	"fmt"
	pluginerr "github.com/gardener/test-infra/pkg/tm-bot/plugins/errors"
	"github.com/spf13/pflag"
	"io"
	"strings"

	"github.com/gardener/test-infra/pkg/tm-bot/github"
)

// Plugin specifies a tm github bot plugin/command that can be triggered by a user
type Plugin interface {
	// Command returns the unique matching command for the plugin
	Command() string

	// Flags return command line style flags for the command
	Flags() *pflag.FlagSet

	// Run runs the command with the parsed flags (flag.Parse()) and the event that triggered the command
	Run(fs *pflag.FlagSet, client github.Client, event *github.GenericRequestEvent) error

	// Description returns a short description of the plugin
	Description() string

	// Example returns an example for the command
	Example() string
}

var Plugins = make(plugins, 0)

type plugins map[string]Plugin

func Register(plugin Plugin) {
	Plugins[plugin.Command()] = plugin
}

func (p *plugins) Get(name string) (Plugin, error) {
	plugin, ok := Plugins[name]
	if !ok {
		return nil, pluginerr.New(fmt.Sprintf("no plugin found for %s", name), fmt.Sprintf("no plugin found for %s", name))
	}
	return plugin, nil
}

func HandleRequest(client github.Client, event *github.GenericRequestEvent) error {
	commands, err := ParseCommands(event.GetMessage())
	if err != nil {
		return pluginerr.Wrap(err, "Internal parse error")
	}

	for _, args := range commands {
		plugin, err := Plugins.Get(args[0])
		if err != nil {
			return Error(client, event, err)
		}

		fs := plugin.Flags()
		if err := fs.Parse(args[1:]); err != nil {
			return Error(client, event, pluginerr.New(err.Error(), FormatUsageError(args[0], plugin.Description(), plugin.Example(), fs.FlagUsages())))
		}
		if err := plugin.Run(fs, client, event); err != nil {
			return Error(client, event, pluginerr.New(err.Error(), FormatUsageError(args[0], plugin.Description(), plugin.Example(), fs.FlagUsages())))
		}
	}

	return nil
}

// Error responds to the client if an error occurs
func Error(client github.Client, event *github.GenericRequestEvent, err error) error {
	if err := client.Respond(event, FormatErrorResponse(event.GetAuthorName(), pluginerr.ShortForError(err), err.Error())); err != nil {
		return err
	}
	return nil
}

// ParseCommands parses a message and returns a string of commands and arguments
func ParseCommands(message string) ([][]string, error) {
	r := bufio.NewReader(strings.NewReader(message))
	var (
		commands = make([][]string, 0)
		args     []string
		line     string
		err      error
	)
	for {
		line, err = r.ReadString('\n')

		trimmedLine := strings.Trim(line, " \n\t")
		if strings.HasPrefix(trimmedLine, "/") {
			if len(args) != 0 {
				commands = append(commands, args)
			}

			args = strings.Split(trimmedLine, " ")
			args[0] = strings.TrimPrefix(args[0], "/")
		} else if trimmedLine != "" {
			args = append(args, strings.Split(trimmedLine, " ")...)
		}

		if err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
	}

	if len(args) != 0 {
		commands = append(commands, args)
	}

	return commands, nil
}
