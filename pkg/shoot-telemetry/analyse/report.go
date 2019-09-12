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
	Figures []*figures `json:"results"`
}

func (a *report) exportReport(format, path string) error {
	// Check if the report result should be printed to stdout.
	if path == "" {
		if format == common.ReportOutputFormatText {
			fmt.Println(a.getText())
		}
		if format == common.ReportOutputFormatJSON {
			if err := a.printJSON(); err != nil {
				return err
			}
		}
		return nil
	}

	// Write report result to disk.
	if format == common.ReportOutputFormatText {
		if err := a.writeText(path); err != nil {
			return err
		}
	}
	if format == common.ReportOutputFormatJSON {
		if err := a.writeJSON(path); err != nil {
			return err
		}
	}
	return nil
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
	w.Write([]byte(data))
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
