// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package result

import (
	"github.com/go-logr/logr"

	"github.com/gardener/test-infra/pkg/testrunner"
)

// Config represents the configuration for collecting and storing results from a testrun.
type Config struct {
	// OutputDir is the path where the testresults are written to.
	OutputDir string

	// Path to the error directory of concourse to put the notify.cfg in.
	ConcourseOnErrorDir string

	// ComponentDescriptorPath path to the component descriptor file
	ComponentDescriptorPath string

	// Repository specifies the repository used to resolve the component references of the component specified in
	// ComponentDescriptorPath
	Repository string

	// OCMConfigPath is the path to the .ocmconfig file
	// Per default, the file is expected to be at $HOME/.ocmconfig
	OCMConfigPath string

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
