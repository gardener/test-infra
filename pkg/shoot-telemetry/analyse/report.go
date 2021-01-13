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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/common"
)

type report struct {
	Figures []*Figures `json:"results"`
}

func (a *report) exportReport(format, path string) error {
	// Check if the report result should be printed to stdout.
	if path == "" {
		if format == common.ReportOutputFormatText {
			fmt.Println(a.getText())
			return nil
		}
		if format == common.ReportOutputFormatJSON {
			return a.printJSON()
		}
		return fmt.Errorf("unknown format %s", format)
	}

	// Write report result to disk.
	if format == common.ReportOutputFormatText {
		return a.writeText(path)
	}
	if format == common.ReportOutputFormatJSON {
		return a.writeJSON(path)
	}
	return fmt.Errorf("unknown format %s", format)
}

func (a *report) writeJSON(dest string) error {
	outputFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	if err := a.makeJSON(outputFile); err != nil {
		return err
	}
	return nil
}

func (a *report) printJSON() error {
	if err := a.makeJSON(os.Stdout); err != nil {
		return err
	}
	return nil
}

func (a *report) makeJSON(f *os.File) error {
	w := io.Writer(f)
	encoder := json.NewEncoder(w)

	if err := encoder.Encode(a); err != nil {
		return errors.New("JSON enconding failed")
	}
	return nil
}

func (a *report) writeText(dest string) error {
	outputFile, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	data := a.getText()
	w := io.Writer(outputFile)
	if _, err := w.Write([]byte(data)); err != nil {
		return err
	}
	return nil
}

func (a *report) getText() string {
	var sb strings.Builder
	for _, f := range a.Figures {
		sb.WriteString(fmt.Sprintf(`
------
Cluster: %s (provider: %s, region: %s)
Requests:
  Count: %d
  Timeouts: %d`, f.Name, f.Provider, f.Seed, f.CountRequests, f.CountTimeouts))

		if f.ResponseTimeDuration != nil {
			sb.WriteString(fmt.Sprintf(`
  Min response time: %d ms
  Max response time: %d ms
  Avg response time: %.3f ms
  Median response time: %.3f ms
  Standard deviation response time: %.3f ms`, f.ResponseTimeDuration.Min, f.ResponseTimeDuration.Max, f.ResponseTimeDuration.Avg, f.ResponseTimeDuration.Median, f.ResponseTimeDuration.Std))
		}

		if f.DownPeriods != nil {
			sb.WriteString(fmt.Sprintf(`
Downtimes:
  Count: %d
  Min duration: %.3f sec
  Max duration: %.3f sec
  Avg duration: %.3f sec
  Median duration: %.3f sec
  Duration standard deviation: %.3f sec`, f.CountUnhealthyPeriods, f.DownPeriods.Min, f.DownPeriods.Max, f.DownPeriods.Avg, f.DownPeriods.Median, f.DownPeriods.Std))
		}
	}

	return sb.String()
}
