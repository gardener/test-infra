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
	"path/filepath"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/util"
)

func (c *collector) ingestIntoElasticsearch(path string, tr *tmv1beta1.Testrun) error {
	if c.esClient == nil {
		return nil
	}
	if tr.Status.Collected {
		return nil
	}
	if util.DocExists(c.log, c.esClient, tr.Name, tr.Status.StartTime.UTC().Format("2006-01-02T15:04:05Z")) {
		return nil
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("cannot read directory '%s'd: %s", path, err.Error())
	}
	for _, file := range files {
		if !file.IsDir() {
			if err := c.esClient.BulkFromFile(filepath.Join(path, file.Name())); err != nil {
				return err
			}
		}
	}

	tr.Status.Collected = true
	return nil
}
