// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type junitXML struct {
	XMLName    xml.Name  `xml:"testsuites"`
	Testsuites testsuite `xml:"testsuite"`
}

type testsuite struct {
	XMLName   xml.Name  `xml:"testsuite"`
	Duration  duration  `xml:"time,attr"`
	Timestamp timestamp `xml:"timestamp,attr"`
}

type duration time.Duration
type timestamp time.Time

func decodeDuration(s string) (duration, error) {
	d, err := time.ParseDuration(s + "s")
	return duration(d), err
}

func (d *duration) UnmarshalXMLAttr(attr xml.Attr) error {
	d2, err := decodeDuration(attr.Value)
	*d = d2
	return err
}

func decodeTimestamp(s string) (timestamp, error) {
	ts, err := time.Parse(time.RFC3339, fmt.Sprintf("%sZ", s))
	return timestamp(ts), err
}

func (ts *timestamp) UnmarshalXMLAttr(attr xml.Attr) error {
	ts2, err := decodeTimestamp(attr.Value)
	*ts = ts2
	return err
}

func parseJunit(junitFilePath string) (startTime, finishTime time.Time, err error) {
	data, err := os.ReadFile(filepath.Clean(junitFilePath))
	if err != nil {
		return
	}

	junitResults := new(junitXML)
	err = xml.Unmarshal(data, junitResults)
	if err != nil {
		return
	}

	startTime = time.Time(junitResults.Testsuites.Timestamp)
	finishTime = startTime.Add(time.Duration(junitResults.Testsuites.Duration))

	return
}
