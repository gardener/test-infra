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
	ociopts "github.com/gardener/component-cli/ociclient/options"
	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/pkg/shoot-telemetry/analyse"
	"github.com/gardener/test-infra/pkg/testrunner"
	telemetryCtrl "github.com/gardener/test-infra/pkg/testrunner/telemetry"
)

// Config represents the configuration for collecting and storing results from a testrun.
type Config struct {
	// OutputDir is the path where the testresults are written to.
	OutputDir string

	// Path to the error directory of concourse to put the notify.cfg in.
	ConcourseOnErrorDir string

	// EnableTelemetry enables the measurement of shoot downtimes during execution
	EnableTelemetry bool

	// ComponentDescriptorPath path to the component descriptor file
	ComponentDescriptorPath string

	// OCIOpts describe options to build a oci client
	OCIOpts *ociopts.Options

	// AssetComponents indicates to upload the testrun status to the github component as an asset
	AssetComponents []string

	// GithubUser to allow getting a github client
	GithubUser string

	GithubPassword string

	// UploadStatusAsset states whether the testrun status should be uploaded as a github release asset
	UploadStatusAsset bool

	// AssetPrefix defines the asset name prefix
	AssetPrefix string

	// ConcourseURL provides the concourse URL as reference for testruns summary results
	ConcourseURL string

	// GrafanaURL provides the grafana dashboard URL for test results
	GrafanaURL string

	// SlackToken is the slack token for slack API operations
	SlackToken string

	// SlackChannel defines in which slack channel the testruns summary shall be posted
	SlackChannel string

	// PostSummaryInSlack states whether the summary of testruns shall be posted in slack
	PostSummaryInSlack bool
}

type Collector struct {
	log            logr.Logger
	config         Config
	kubeconfigPath string

	telemetry        *telemetryCtrl.Telemetry
	telemetryResults map[string]*analyse.Figures

	// RunExecCh is called when a new testrun is executed
	RunExecCh chan *testrunner.Run
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
