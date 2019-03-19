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

package result

import (
	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config represents the configuration for collecting and storing results from a testrun.
type Config struct {
	// OutputFilePath is the path where the testresult is written to.
	OutputFile string

	// Config name of the elasticsearch instance to store the test results.
	ESConfigName string

	// Endpoint of the s3 storage of the testmachinery.
	S3Endpoint string

	// S3SSL indicates whether the S3 instance is SSL secured or not.
	S3SSL bool

	// Path to the error directory of concourse to put the notify.cfg in.
	ConcourseOnErrorDir string
}

// SummaryType defines the type of a test result or summary
type SummaryType string

// Summary types can be testrun or teststep
const (
	SummaryTypeTestrun  SummaryType = "testrun"
	SummaryTypeTeststep SummaryType = "teststep"
)

// Metadata is the common metadata of all ouputs and summaries.
type Metadata struct {
	// Landscape describes the current dev,staging,canary,office or live.
	Landscape         string `json:"landscape"`
	CloudProvider     string `json:"cloudprovider"`
	KubernetesVersion string `json:"k8s_version"`

	// ComponentDescriptor describes the current component_descriptor of the direct landscape-setup components.
	// It is formated as an array of components: { name: "my_component", version: "0.0.1" }
	ComponentDescriptor []*componentdescriptor.Component `json:"bom"`
	TestrunID           string                           `json:"testrun_id"`
}

// StepExportMetadata is the metadata of one step of a testrun.
type StepExportMetadata struct {
	Metadata
	TestDefName string           `json:"testdefinition"`
	Phase       argov1.NodePhase `json:"phase,omitempty"`
	StartTime   *metav1.Time     `json:"startTime,omitempty"`
	Duration    int64            `json:"duration,omitempty"`
}

// TestrunSummary is the result of the overall testrun.
type TestrunSummary struct {
	Metadata  *Metadata        `json:"tm"`
	Type      SummaryType      `json:"type"`
	Phase     argov1.NodePhase `json:"phase,omitempty"`
	StartTime *metav1.Time     `json:"startTime,omitempty"`
	Duration  int64            `json:"duration,omitempty"`
	TestsRun  int              `json:"testsRun,omitempty"`
}

// StepSummary is the result of a specific step.
type StepSummary struct {
	Metadata  *Metadata        `json:"tm"`
	Type      SummaryType      `json:"type"`
	Name      string           `json:"name,omitempty"`
	Phase     argov1.NodePhase `json:"phase,omitempty"`
	StartTime *metav1.Time     `json:"startTime,omitempty"`
	Duration  int64            `json:"duration,omitempty"`
}

// notificationConfig is the configuration that is used by concourse to send notifications.
type notificationCfg struct {
	Email email `yaml:"email"`
}

type email struct {
	Subject    string   `yaml:"subject"`
	Recipients []string `yaml:"recipients"`
	MailBody   string   `yaml:"mail_body"`
}
