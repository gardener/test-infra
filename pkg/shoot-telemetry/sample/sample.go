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

package sample

import "time"

// Sample represent one measurement.
type Sample struct {
	ResponseDuration time.Duration
	Status           int
	Timestamp        time.Time
}

// NewSample returns a pointer to a new sample.
func NewSample(statusCode int, timestamp time.Time) *Sample {
	// TODO Should we directly convert the timestamp into a string with the proper format?
	return &Sample{
		ResponseDuration: time.Now().Sub(timestamp),
		Status:           statusCode,
		Timestamp:        timestamp,
	}
}
