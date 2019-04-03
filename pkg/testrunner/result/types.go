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

	// Endpoint of the argo ui of the testmachinery.
	ArgoUIEndpoint string

	// Endpoint of kibana for the logs of the testmachinery.
	KibanaEndpoint string

	// Path to the error directory of concourse to put the notify.cfg in.
	ConcourseOnErrorDir string
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

type kibanaFilter struct {
	IndexPatternID string
	WorkflowID     string
	PodID          string
}
