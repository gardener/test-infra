// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"os"
	"os/exec"

	"github.com/go-logr/logr"
)

// RunCommand constructs a command, logs it with all arguments and executes it.
func RunCommand(log logr.Logger, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	log.Info(cmd.String())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
