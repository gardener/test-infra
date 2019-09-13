// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package controller

import (
	"encoding/csv"
	"os"
	"strconv"
	"time"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
	"github.com/gardener/test-infra/pkg/shoot-telemetry/sample"
	log "github.com/sirupsen/logrus"
)

func (c *controller) generateOutput() error {
	outputFile, err := os.OpenFile(c.config.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	var (
		doc    = csv.NewWriter(outputFile)
		record = []string{common.MeasurementsHeadCluster, common.MeasurementsHeadProvider, common.MeasurementsHeadSeed, common.MeasurementsHeadTimestamp, common.MeasurementsHeadStatusCode, common.MeasurementsHeadResponseTime}
	)
	log.Debugf("Write measurements to file")

	outputFileStat, err := outputFile.Stat()
	if err != nil {
		return err
	}
	// If file size is zero then it seems to be a new file and we would write the csv head row.
	if outputFileStat.Size() == 0 {
		if err := doc.Write(record); err != nil {
			return err
		}
	}

	// Add an entry for every sample.
	c.targetsMutex.Lock()
	for key, target := range c.targets {
		for _, sample := range target.series {
			record[0] = key
			record[1] = target.provider
			record[2] = target.seedName
			record[3] = sample.Timestamp.Format(time.RFC3339)
			record[4] = strconv.Itoa(sample.Status)
			record[5] = strconv.FormatInt(sample.ResponseDuration.Nanoseconds()/1e6, 10)
			doc.Write(record)
		}
	}

	// Write the data to disk and forget the old targets.
	doc.Flush()
	if err := doc.Error(); err != nil {
		return err
	}

	// Reset the in memory state.
	for key := range c.targets {
		c.targets[key].series = []*sample.Sample{}
	}
	c.targetsMutex.Unlock()

	return nil
}
