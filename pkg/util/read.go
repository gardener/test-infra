// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bufio"
	"bytes"
	"io"
)

// ReadLines reads a byte array line by line ('\n' or '\r\n'); and return the content without the line end.
func ReadLines(document []byte) <-chan []byte {
	c := make(chan []byte)
	go func() {
		reader := bufio.NewReader(bytes.NewReader(document))
		doc := make([]byte, 0)
		for {
			line, isPrefix, err := reader.ReadLine()
			if err == io.EOF {
				break
			}
			if err != nil {
				return
			}
			doc = append(doc, line...)
			if isPrefix {
				continue
			}
			c <- doc
			doc = make([]byte, 0)
		}

		close(c)
	}()
	return c
}
