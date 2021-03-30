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

package collector

import (
	"fmt"
	"io/ioutil"
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
		if err := ioutil.WriteFile(file, buf, 0644); err != nil {
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
