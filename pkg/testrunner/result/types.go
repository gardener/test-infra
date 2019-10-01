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
	telemetryCtrl "github.com/gardener/test-infra/pkg/testrunner/telemetry"
	"github.com/go-logr/logr"
)

// Config represents the configuration for collecting and storing results from a testrun.
type Config struct {
	// OutputDir is the path where the testresults are written to.
	OutputDir string

	// Config name of the elasticsearch instance to store the test results.
	ESConfigName string

	// Endpoint of the s3 storage of the testmachinery.
	S3Endpoint string

	// S3SSL indicates whether the S3 instance is SSL secured or not.
	S3SSL bool

	// Path to the error directory of concourse to put the notify.cfg in.
	ConcourseOnErrorDir string

	// EnableTelemetry enbales the measurement of shoot downtimes during execution
	EnableTelemetry bool

	// ComponentDescriptorPath path to the component descriptor file
	ComponentDescriptorPath string

	// GithubComponentForStatus indicates to upload the testrun status to the github component as an asset
	GithubComponentForStatus string

	// GithubUser to allow getting a github client
	GithubUser string

	GithubPassword    string

	// UploadStatusAsset states whether the testrun status should be uploaded as a github release asset
	UploadStatusAsset bool
}

type Collector struct {
	log       logr.Logger
	config    Config
	telemetry *telemetryCtrl.Telemetry
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
