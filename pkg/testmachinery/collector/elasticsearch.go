// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	"fmt"
	"os"
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

	files, err := os.ReadDir(path)
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
