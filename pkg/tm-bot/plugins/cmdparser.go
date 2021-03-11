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

package plugins

import (
	"bufio"
	"github.com/kballard/go-shellquote"
	"io"
	"strings"
)

// ParseCommands parses a message and returns a string of commands and arguments
func ParseCommands(message string) ([][]string, error) {
	// first replace possible line breaks with \n
	message = strings.ReplaceAll(message, "\r\n", "\n")
	message = strings.ReplaceAll(message, "<br>", "\n")
	message = strings.ReplaceAll(message, "</br>", "\n")
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
			args, err := shellquote.Split(trimmedLine)
			if err != nil {
				continue
			}
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
