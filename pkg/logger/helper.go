// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

func Logf(logFunc func(msg string, keysAndValues ...interface{}), format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	logFunc(message)
}

var (
	ghaSummaryFileName   string
	ghaSummaryFileExists bool
	fileWriterMutex      sync.Mutex
	setupOnce            sync.Once
)

func SetupGitHubStepSummary(file string) {
	onceFunc := func() {
		if file != "" {
			ghaSummaryFileExists = true
			ghaSummaryFileName = filepath.Clean(file)
		}
	}
	setupOnce.Do(onceFunc)
}

func PostToGitHubStepSummary(message string, append bool) error {
	if ghaSummaryFileExists {
		fileWriterMutex.Lock()
		defer fileWriterMutex.Unlock()

		var flags int
		if append {
			flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
		} else {
			flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
		}
		file, err := os.OpenFile(ghaSummaryFileName, flags, 0600) // #nosec G304 -- input is derived form a user's input
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				fmt.Printf("Could not close file %s: %v\n", ghaSummaryFileName, err)
			}
		}(file)
		if err != nil {
			return err
		}

		log := []byte(message + "\n")
		_, err = file.Write(log)
		return err
	}
	return nil
}
