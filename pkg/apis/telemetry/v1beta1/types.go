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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

////////////////////////////////////////////////////
//               ShootsMeasurement                   //
////////////////////////////////////////////////////

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Testrun is the description of the testflow that should be executed.
// +k8s:openapi-gen=true
type ShootsMeasurement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ShootsMeasurementSpec   `json:"spec"`
	Status ShootsMeasurementStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestrunList contains a list of Testruns
type ShootsMeasurementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ShootsMeasurement `json:"items"`
}

// ShootsMeasurementSpec is the specification of a measurement of shoots.
type ShootsMeasurementSpec struct {
	// Secret reference to the gardener kubernetes cluster where the shoots to watch reside
	GardenerSecretRef string `json:"gardenerSecretRef,omitempty"`

	// Shoots specify a list of shoots to watch
	Shoots []client.ObjectKey `json:"shoots,omitempty"`
}

// ShootsMeasurementStatus is the status of the Shoots telemetry.
type ShootsMeasurementStatus struct {
	// Controller specifies the telemetry controller that handles the data
	Controller string `json:"controller,omitempty"`

	// Phase indicates the current state of the telemetry measurement
	Phase TelemetryPhase `json:"phase,omitempty"`

	// Message indicates current failures of the measurement
	Message string `json:"message,omitempty"`

	// Data specifies the result of the monitored shoots
	Data []ShootMeasurementData `json:"data,omitempty"`

	// ObservedGeneration is the most recent generation observed for this testrun.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ShootMeasurementData is the telemetry result of one shoot
type ShootMeasurementData struct {
	Shoot                 client.ObjectKey `json:"shoot"`
	Provider              string           `json:"provider"`
	Seed                  string           `json:"seed"`
	CountUnhealthyPeriods int              `json:"countUnhealthyPeriods"`
	CountRequests         int              `json:"countRequest"`
	CountTimeouts         int              `json:"countRequestTimeouts"`

	// +optional
	DownPeriods *DowntimePeriods `json:"downTimesSec,omitempty"`

	// +optional
	ResponseTimeDuration *ResponseTimeDuration `json:"responseTimesMs,omitempty"`
}

type ResponseTimeDuration struct {
	Min    int     `json:"min"`
	Max    int     `json:"max"`
	Avg    float64 `json:"avg"`
	Median float64 `json:"median"`
	Std    float64 `json:"std"`
}

type DowntimePeriods struct {
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Avg    float64 `json:"avg"`
	Median float64 `json:"median"`
	Std    float64 `json:"std"`
}
