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

package analyse

import (
	"encoding/csv"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
)

// Analyse reads a file with measurements on a given path <inputFilePath> and
// detects periods when Cluster API servers were not healthy. It calculates and
// prints some statistical key figures for the unhealthy periods per cluster.
// The <outputPath> parameter specifies the file to store the analysis. Empty string means stdout.
// The <outputFormat> parameter specifies how the analysis results should be formatted.
func Analyse(inputFilePath, outputPath, outputFormat string) error {
	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		return errors.New("input file does not exist")
	}
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	var (
		csvReader     = csv.NewReader(inputFile)
		downTimeCache = make(map[string]string)
		figuresStore  = make(map[string]*figures)
		rowCounter    int
	)

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if len(record) != 6 {
			return fmt.Errorf("invalid row %d", rowCounter)
		}
		// Skip the first/head row.
		if record[0] == common.MeasurementsHeadCluster && record[1] == common.MeasurementsHeadProvider && record[2] == common.MeasurementsHeadSeed && record[3] == common.MeasurementsHeadTimestamp && record[4] == common.MeasurementsHeadStatusCode && record[5] == common.MeasurementsHeadResponseTime {
			rowCounter++
			continue
		}

		// Check if a figure for this entry already exists, if not create one.
		figure, exists := figuresStore[record[0]]
		if !exists {
			figure = &figures{
				Name:     record[0],
				Provider: record[1],
				Seed:     record[2],
			}
			figuresStore[record[0]] = figure
		}

		// Parse Request Duration
		responseTime, err := strconv.Atoi(record[5])
		if err != nil {
			return err
		}
		// Ignore timeouts and count the request timeout occurrences.
		figure.CountRequests++
		if responseTime < common.RequestTimeOut {
			figure.requestDurationStore = append(figure.requestDurationStore, &responseTime)
		} else {
			figure.CountTimeouts++
		}

		// Parse Status Code.
		statusCode, err := strconv.Atoi(record[4])
		if err != nil {
			return err
		}
		downTimeStart, cached := downTimeCache[record[0]]
		if statusCode >= 200 && statusCode < 299 {
			if cached {
				downTimeStart, err := time.Parse(time.RFC3339, downTimeStart)
				if err != nil {
					return err
				}
				downTimeEnd, err := time.Parse(time.RFC3339, record[3])
				if err != nil {
					return err
				}
				figure.CountUnhealthyPeriods++
				figure.downPeriodsStore = append(figure.downPeriodsStore, downTimeEnd.Sub(downTimeStart))
				delete(downTimeCache, record[0])
			}
			rowCounter++
			continue
		}

		// If unhealthy status code and not cached yet means beginn of a new downtime period.
		if !cached {
			downTimeCache[record[0]] = record[3]
		}
		rowCounter++
	}

	// Create a new report.
	result := report{
		Figures: []*figures{},
	}

	// Calculate statistical figures per cluster and add them to the report.
	for _, f := range figuresStore {
		f.calculateDownPeriodStatistics()
		f.calculateResponseTimeStatistics()
		result.Figures = append(result.Figures, f)
	}

	// Export the report.
	if err := result.exportReport(outputFormat, outputPath); err != nil {
		return err
	}

	return nil
}
