// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/util"
)

func writeBulks(path string, bufs [][]byte) error {
	// check if directory exists and create of not
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	for _, buf := range bufs {
		file := filepath.Join(path, fmt.Sprintf("res-%s", util.RandomString(5)))
		if err := os.WriteFile(file, buf, 0644); err != nil {
			return err
		}
	}
	return nil
}

func marshalAndAppendSummaries(summary metadata.TestrunSummary, stepSummaries []metadata.StepSummary) ([][]byte, error) {
	b, err := util.MarshalNoHTMLEscape(summary)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal %s", err.Error())
	}

	s := [][]byte{b}
	for _, summary := range stepSummaries {
		b, err := util.MarshalNoHTMLEscape(summary)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal %s", err.Error())
		}
		s = append(s, b)
	}
	return s, nil

}
