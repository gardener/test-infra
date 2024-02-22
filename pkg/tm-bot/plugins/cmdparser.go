// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package plugins

import (
	"bufio"
	"io"
	"strings"

	"github.com/kballard/go-shellquote"
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
