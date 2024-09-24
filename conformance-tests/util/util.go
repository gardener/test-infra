package util

import (
	"os"
	"os/exec"

	"github.com/go-logr/logr"
)

func RunCommand(log logr.Logger, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	log.Info(cmd.String())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
