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

package testrunner

import (
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/watch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

// Config are configuration of the environment like the testmachinery cluster or S3 store
// where the testrunner executes the testrun.
type Config struct {
	// Testrun watch controller
	Watch watch.Watch

	// Namespace where the testrun is deployed.
	Namespace string

	// Max wait time for a testrun to finish.
	Timeout time.Duration

	// Number of testrun retries after a failed run
	FlakeAttempts int

	ExecutorConfig
}

// RunEventFunc is called every time a new testrun is triggered
// Also notifies for retries
type RunEventFunc func(run *Run)

// Run describes a testrun that is executed by the testrunner.
// It consists of a testrun and its metadata
type Run struct {
	// Specify internal info for specific run types
	Info     interface{}
	Testrun  *tmv1beta1.Testrun
	Metadata *Metadata
	Error    error

	Rerenderer Rerenderer
}

// Rerenderer is instance that rerenders the current run to make it retryable.
type Rerenderer interface {
	Rerender(tr *tmv1beta1.Testrun) (*Run, error)
}

// RunList represents a list of Runs.
type RunList []*Run

// SummaryType defines the type of a test result or summary
type SummaryType string

// Summary types can be testrun or teststep
const (
	SummaryTypeTestrun  SummaryType = "testrun"
	SummaryTypeTeststep SummaryType = "teststep"
)

// Metadata is the common metadata of all outputs and summaries.
type Metadata struct {
	// Short description of the flavor
	FlavorDescription string `json:"flavor_description,omitempty"`

	// Landscape describes the current dev,staging,canary,office or live.
	Landscape         string `json:"landscape,omitempty"`
	CloudProvider     string `json:"cloudprovider,omitempty"`
	KubernetesVersion string `json:"k8s_version,omitempty"`
	OperatingSystem   string `json:"operating_system,omitempty"`
	Region            string `json:"region,omitempty"`
	Zone              string `json:"zone,omitempty"`

	// ComponentDescriptor describes the current component_descriptor of the direct landscape-setup components.
	// It is formatted as an array of components: { name: "my_component", version: "0.0.1" }
	ComponentDescriptor interface{} `json:"bom,omitempty"`

	// UpgradedComponentDescriptor describes the updated component_descriptor.
	// It is formatted as an array of components: { name: "my_component", version: "0.0.1" }
	UpgradedComponentDescriptor interface{} `json:"upgraded_bom,omitempty"`

	// Name of the testrun crd object.
	Testrun TestrunMetadata `json:"tr"`

	// all environment configuration values
	Configuration map[string]string `json:"config,omitempty"`

	// Additional annotations form the testrun or steps
	Annotations map[string]string `json:"annotations,omitempty"`

	// Represents how many retries the testrun had
	Retries int `json:"retries,omitempty"`

	// Contains the measured telemetry data
	// Is only used for internal sharing.
	TelemetryData *TelemetryData `json:"-"`
}

// TestrunMetadata represents the metadata of a testrun
type TestrunMetadata struct {
	// Name of the testrun crd object.
	ID string `json:"id"`

	// ID of the execution group this test belongs to
	ExecutionGroup string `json:"executionGroup,omitempty"`

	// StartTime of the testrun.
	StartTime *metav1.Time `json:"startTime"`
}

// StepExportMetadata is the metadata of one step of a testrun.
type StepExportMetadata struct {
	Metadata
	StepName    string           `json:"stepName,omitempty"`
	TestDefName string           `json:"testdefinition,omitempty"`
	Phase       argov1.NodePhase `json:"phase,omitempty"`
	StartTime   *metav1.Time     `json:"startTime,omitempty"`
	Duration    int64            `json:"duration,omitempty"`
	PodName     string           `json:"podName"`
}

// TestrunSummary is the result of the overall testrun.
type TestrunSummary struct {
	Metadata      *Metadata        `json:"tm,omitempty"`
	Type          SummaryType      `json:"type,omitempty"`
	Phase         argov1.NodePhase `json:"phase,omitempty"`
	StartTime     *metav1.Time     `json:"startTime,omitempty"`
	Duration      int64            `json:"duration,omitempty"`
	TestsRun      int              `json:"testsRun,omitempty"`
	TelemetryData *TelemetryData   `json:"telemetry,omitempty"`
}

// StepSummary is the result of a specific step.
type StepSummary struct {
	Metadata    *Metadata        `json:"tm,omitempty"`
	Type        SummaryType      `json:"type,omitempty"`
	Name        string           `json:"name,omitempty"`
	StepName    string           `json:"stepName,omitempty"`
	Labels      []string         `json:"labels,omitempty"`
	Phase       argov1.NodePhase `json:"phase,omitempty"`
	StartTime   *metav1.Time     `json:"startTime,omitempty"`
	Duration    int64            `json:"duration,omitempty"`
	PreComputed *StepPreComputed `json:"pre,omitempty"`
}

// StepPreComputed contains fields that could be created at runtime via scripted fields, but are created statically for better performance and better support of grafana
type StepPreComputed struct {
	// same as StepSummary.Phase but mapping states to ints (Failed&Timeout -> 0, Succeeded -> 100); allows to do averages on success rate in dashboards
	PhaseNum *int `json:"phaseNum,omitempty"`
	// A K8S Version without the patch suffix, e.g. "1.16"
	K8SMajorMinorVersion string `json:"k8sMajMinVer,omitempty"`
	// Dummy field for grafana/log links
	LogsDisplayName string `json:"logsText,omitempty"`
	// Dummy field for argoui/workflow links
	ArgoDisplayName string `json:"argoText,omitempty"`
	// the cluster domain of the testmachinery (useful to build other URLs in dashboards)
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

// Dimension describes the basic dimension of a test
type Dimension struct {
	Description       string `json:"description,omitempty"`
	Cloudprovider     string `json:"cloudprovider,omitempty"`
	KubernetesVersion string `json:"k8sVersion,omitempty"`
	OperatingSystem   string `json:"operating_system,omitempty"`
}

// TelemetryData describes the measured telemetry data for the tested shoot
type TelemetryData struct {
	ResponseTime    *TelemetryResponseTimeDuration `json:"response_time,omitempty"`
	DowntimePeriods *TelemetryDowntimePeriods      `json:"downtime,omitempty"`
}

// TelemetryResponseTimeDuration describes the response data of the telemetry measurement
type TelemetryResponseTimeDuration struct {
	Min    int   `json:"min"`
	Max    int   `json:"max"`
	Avg    int64 `json:"avg"`
	Median int64 `json:"median"`
	Std    int64 `json:"std"`
}

// TelemetryResponseTimeDuration describes the measured downtimes
type TelemetryDowntimePeriods struct {
	Min    int64 `json:"min"`
	Max    int64 `json:"max"`
	Avg    int64 `json:"avg"`
	Median int64 `json:"median"`
	Std    int64 `json:"std"`
}
